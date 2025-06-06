// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.org/modules/git"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertFileExist(t *testing.T, p string) {
	exist, err := util.IsExist(p)
	require.NoError(t, err)
	assert.True(t, exist)
}

func assertFileEqual(t *testing.T, p string, content []byte) {
	bs, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, content, bs)
}

func TestRepoCloneWiki(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		dstPath := t.TempDir()

		r := fmt.Sprintf("%suser2/repo1.wiki.git", u.String())
		u, _ = url.Parse(r)
		u.User = url.UserPassword("user2", userPassword)
		t.Run("Clone", func(t *testing.T) {
			require.NoError(t, git.CloneWithArgs(t.Context(), git.AllowLFSFiltersArgs(), u.String(), dstPath, git.CloneRepoOptions{}))
			assertFileEqual(t, filepath.Join(dstPath, "Home.md"), []byte("# Home page\n\nThis is the home page!\n"))
			assertFileExist(t, filepath.Join(dstPath, "Page-With-Image.md"))
			assertFileExist(t, filepath.Join(dstPath, "Page-With-Spaced-Name.md"))
			assertFileExist(t, filepath.Join(dstPath, "images"))
			assertFileExist(t, filepath.Join(dstPath, "jpeg.jpg"))
		})
	})
}

func Test_RepoWikiPages(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	url := "/user2/repo1/wiki/?action=_pages"
	req := NewRequest(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body)
	expectedPagePaths := []string{
		"Home", "Long-Page", "Page-With-Image", "Page-With-Spaced-Name", "Unescaped-File", "XSS",
	}
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		firstAnchor := s.Find("a").First()
		href, _ := firstAnchor.Attr("href")
		pagePath := strings.TrimPrefix(href, "/user2/repo1/wiki/")

		assert.Equal(t, expectedPagePaths[i], pagePath)
	})
}
