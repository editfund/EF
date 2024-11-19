// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package e2e

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/require"
)

var rootPathRe = regexp.MustCompile("\\[repository\\]\nROOT\\s=\\s.*")

func onForgejoRunTB(t testing.TB, callback func(testing.TB, *url.URL), prepare ...bool) {
	if len(prepare) == 0 || prepare[0] {
		defer tests.PrepareTestEnv(t, 1)()
	}
	s := http.Server{
		Handler: testE2eWebRoutes,
	}

	u, err := url.Parse(setting.AppURL)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", u.Host)
	i := 0
	for err != nil && i <= 10 {
		time.Sleep(100 * time.Millisecond)
		listener, err = net.Listen("tcp", u.Host)
		i++
	}
	require.NoError(t, err)
	u.Host = listener.Addr().String()

	// Override repository root in config.
	conf, err := os.ReadFile(setting.CustomConf)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(setting.CustomConf, rootPathRe.ReplaceAll(conf, []byte("[repository]\nROOT = "+setting.RepoRootPath)), 0o644))

	defer func() {
		require.NoError(t, os.WriteFile(setting.CustomConf, conf, 0o644))
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		s.Shutdown(ctx)
		cancel()
	}()

	go s.Serve(listener)
	// Started by config go ssh.Listen(setting.SSH.ListenHost, setting.SSH.ListenPort, setting.SSH.ServerCiphers, setting.SSH.ServerKeyExchanges, setting.SSH.ServerMACs)

	callback(t, u)
}

func onForgejoRun(t *testing.T, callback func(*testing.T, *url.URL), prepare ...bool) {
	onForgejoRunTB(t, func(t testing.TB, u *url.URL) {
		callback(t.(*testing.T), u)
	}, prepare...)
}
