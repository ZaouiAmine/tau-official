package auth

import (
	"context"
	"testing"

	"github.com/taubyte/tau/pkg/kvdb/mock"
	"gotest.tools/v3/assert"
)

// Registering over p2p has no GitHub client, so it cannot install a deploy key:
// replacing a stored key there leaves one GitHub never heard of, and every clone
// then fails with "Repository not found".
func TestP2PRegisterKeepsARepositorysDeployKey(t *testing.T) {
	ctx := context.Background()
	db, err := mock.New().New(nil, "test", 0)
	assert.NilError(t, err)
	srv := &AuthService{db: db, devMode: true, tnsClient: &mockTNSClient{}}

	key := "/repositories/github/123/key"
	assert.NilError(t, db.Put(ctx, key, []byte("the key GitHub has")))

	response, err := srv.registerGitHubRepository(ctx, nil, "123")
	assert.NilError(t, err)
	assert.Equal(t, response.Key, key)

	stored, err := db.Get(ctx, key)
	assert.NilError(t, err)
	assert.Equal(t, string(stored), "the key GitHub has")
}
