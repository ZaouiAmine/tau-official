package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/taubyte/tau/core/kvdb"
	"github.com/taubyte/tau/pkg/kvdb/mock"
	"github.com/taubyte/tau/services/auth/projects"
	"gotest.tools/v3/assert"
)

type countingKV struct {
	kvdb.KVDB
	gets atomic.Int32
}

func (c *countingKV) Get(ctx context.Context, key string) ([]byte, error) {
	c.gets.Add(1)
	return c.KVDB.Get(ctx, key)
}

type listedRepos struct {
	*mockGitHubClient
	repos map[string]RepositoryBasicInfo
}

func (l listedRepos) ListMyRepos() (map[string]RepositoryBasicInfo, error) { return l.repos, nil }

// Listing reads each project once and only its name: a project has two
// repositories, and reading it whole for each cost five reads per repository.
func TestProjectListingReadsEachProjectOnce(t *testing.T) {
	ctx := context.Background()
	db, err := mock.New().New(nil, "test", 0)
	assert.NilError(t, err)
	kv := &countingKV{KVDB: db}
	srv := &AuthService{db: kv}

	const n = 50
	repos := make(map[string]RepositoryBasicInfo)
	for i := range n {
		id := fmt.Sprintf("p%d", i)
		project, err := projects.New(kv, projects.Data{"id": id, "name": id, "provider": "github", "config": fmt.Sprint(2 * i), "code": fmt.Sprint(2*i + 1)})
		assert.NilError(t, err)
		assert.NilError(t, project.Register())
		for _, repo := range []string{fmt.Sprint(2 * i), fmt.Sprint(2*i + 1)} {
			assert.NilError(t, kv.Put(ctx, "/repositories/github/"+repo+"/project", []byte(id)))
			repos[repo] = RepositoryBasicInfo{}
		}
	}

	response, err := srv.getGitHubUserProjects(ctx, listedRepos{&mockGitHubClient{}, repos})
	assert.NilError(t, err)
	assert.Equal(t, len(response.Projects), n)
	// one lookup per repository, one name per project
	assert.Equal(t, int(kv.gets.Load()), 3*n)
}

// Listings that overlap, as parallel sessions on one token do, share one walk
// of GitHub's pages.
func TestOverlappingListingsShareOneWalk(t *testing.T) {
	var calls atomic.Int32
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Link", fmt.Sprintf(`<%s/user/repos?page=3>; rel="last"`, srv.URL))
		json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "full_name": "me/r", "url": "u"}})
	}))
	defer srv.Close()

	gh := github.NewClient(nil)
	gh.BaseURL, _ = url.Parse(srv.URL + "/")
	client := &githubClient{Client: gh, Token: "overlapping-listings-token", ctx: context.Background()}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := client.ListMyRepos()
			assert.Check(t, err == nil)
		}()
	}
	wg.Wait()
	assert.Equal(t, int(calls.Load()), 3, "three pages, once")
}
