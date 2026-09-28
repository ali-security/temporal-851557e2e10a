package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseArtifactName(t *testing.T) {
	parsed, ok := ParseArtifactName("junit-xml--22373551837--64609560060--2--integration-0--Integration--functional-test")
	require.True(t, ok)
	require.Equal(t, ArtifactName{
		Type:       "junit-xml",
		RunID:      "22373551837",
		JobID:      "64609560060",
		RunAttempt: 2,
		NameSuffix: "integration-0--Integration--functional-test",
	}, parsed)

	parsed, ok = ParseArtifactName("junit-xml--1--2--invalid--mysql8--shard0--functional-test")
	require.False(t, ok)
	require.Equal(t, ArtifactName{
		Type:       "junit-xml",
		RunID:      "1",
		JobID:      "2",
		NameSuffix: "mysql8--shard0--functional-test",
	}, parsed)

	_, ok = ParseArtifactName("test-results")
	require.False(t, ok)
}

func TestDownloadArtifactRetriesIncompleteResponses(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			writer.Header().Set("Content-Length", "8")
			_, _ = writer.Write([]byte("bad"))
			return
		}
		_, _ = writer.Write([]byte("complete"))
	}))
	defer server.Close()

	restoreAPIClient(t, server.URL, server.Client())
	t.Setenv("GH_TOKEN", "test-token")
	outputDir := t.TempDir()

	zipPath, err := downloadArtifactWithRetry(
		context.Background(),
		"temporalio/temporal",
		42,
		outputDir,
		time.Nanosecond,
	)
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
	content, err := os.ReadFile(zipPath)
	require.NoError(t, err)
	require.Equal(t, "complete", string(content))
	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, filepath.Base(zipPath), entries[0].Name())
}
