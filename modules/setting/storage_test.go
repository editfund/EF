// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getStorageMultipleName(t *testing.T) {
	iniStr := `
[lfs]
MINIO_BUCKET = gitea-lfs

[attachment]
MINIO_BUCKET = gitea-attachment

[storage]
STORAGE_TYPE = minio
MINIO_BUCKET = gitea-storage
`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)

	require.NoError(t, loadAttachmentFrom(cfg))
	assert.Equal(t, "gitea-attachment", Attachment.Storage.MinioConfig.Bucket)
	assert.Equal(t, "attachments/", Attachment.Storage.MinioConfig.BasePath)

	require.NoError(t, loadLFSFrom(cfg))
	assert.Equal(t, "gitea-lfs", LFS.Storage.MinioConfig.Bucket)
	assert.Equal(t, "lfs/", LFS.Storage.MinioConfig.BasePath)

	require.NoError(t, loadAvatarsFrom(cfg))
	assert.Equal(t, "gitea-storage", Avatar.Storage.MinioConfig.Bucket)
	assert.Equal(t, "avatars/", Avatar.Storage.MinioConfig.BasePath)
}

func Test_getStorageUseOtherNameAsType(t *testing.T) {
	iniStr := `
[attachment]
STORAGE_TYPE = lfs

[storage.lfs]
STORAGE_TYPE = minio
MINIO_BUCKET = gitea-storage
`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)

	require.NoError(t, loadAttachmentFrom(cfg))
	assert.Equal(t, "gitea-storage", Attachment.Storage.MinioConfig.Bucket)
	assert.Equal(t, "attachments/", Attachment.Storage.MinioConfig.BasePath)

	require.NoError(t, loadLFSFrom(cfg))
	assert.Equal(t, "gitea-storage", LFS.Storage.MinioConfig.Bucket)
	assert.Equal(t, "lfs/", LFS.Storage.MinioConfig.BasePath)
}

func Test_getStorageInheritStorageType(t *testing.T) {
	iniStr := `
[storage]
STORAGE_TYPE = minio
`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)

	require.NoError(t, loadPackagesFrom(cfg))
	assert.EqualValues(t, "minio", Packages.Storage.Type)
	assert.Equal(t, "gitea", Packages.Storage.MinioConfig.Bucket)
	assert.Equal(t, "packages/", Packages.Storage.MinioConfig.BasePath)

	require.NoError(t, loadRepoArchiveFrom(cfg))
	assert.EqualValues(t, "minio", RepoArchive.Storage.Type)
	assert.Equal(t, "gitea", RepoArchive.Storage.MinioConfig.Bucket)
	assert.Equal(t, "repo-archive/", RepoArchive.Storage.MinioConfig.BasePath)

	require.NoError(t, loadActionsFrom(cfg))
	assert.EqualValues(t, "minio", Actions.LogStorage.Type)
	assert.Equal(t, "gitea", Actions.LogStorage.MinioConfig.Bucket)
	assert.Equal(t, "actions_log/", Actions.LogStorage.MinioConfig.BasePath)

	assert.EqualValues(t, "minio", Actions.ArtifactStorage.Type)
	assert.Equal(t, "gitea", Actions.ArtifactStorage.MinioConfig.Bucket)
	assert.Equal(t, "actions_artifacts/", Actions.ArtifactStorage.MinioConfig.BasePath)

	require.NoError(t, loadAvatarsFrom(cfg))
	assert.EqualValues(t, "minio", Avatar.Storage.Type)
	assert.Equal(t, "gitea", Avatar.Storage.MinioConfig.Bucket)
	assert.Equal(t, "avatars/", Avatar.Storage.MinioConfig.BasePath)

	require.NoError(t, loadRepoAvatarFrom(cfg))
	assert.EqualValues(t, "minio", RepoAvatar.Storage.Type)
	assert.Equal(t, "gitea", RepoAvatar.Storage.MinioConfig.Bucket)
	assert.Equal(t, "repo-avatars/", RepoAvatar.Storage.MinioConfig.BasePath)
}

type testLocalStoragePathCase struct {
	loader       func(rootCfg ConfigProvider) error
	storagePtr   **Storage
	expectedPath string
}

func testLocalStoragePath(t *testing.T, appDataPath, iniStr string, cases []testLocalStoragePathCase) {
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	AppDataPath = appDataPath
	for _, c := range cases {
		require.NoError(t, c.loader(cfg))
		storage := *c.storagePtr

		assert.EqualValues(t, "local", storage.Type)
		assert.True(t, filepath.IsAbs(storage.Path))
		assert.Equal(t, filepath.Clean(c.expectedPath), filepath.Clean(storage.Path))
	}
}

func Test_getStorageInheritStorageTypeLocal(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/repo-archive"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPath(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = /data/gitea
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/data/gitea/attachments"},
		{loadLFSFrom, &LFS.Storage, "/data/gitea/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/data/gitea/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/data/gitea/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/repo-archive"},
		{loadActionsFrom, &Actions.LogStorage, "/data/gitea/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/data/gitea/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/data/gitea/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalRelativePath(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = storages
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/storages/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/storages/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/storages/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/storages/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/storages/repo-archive"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/storages/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/storages/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/storages/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = /data/gitea

[repo-archive]
PATH = /data/gitea/the-archives-dir
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/data/gitea/attachments"},
		{loadLFSFrom, &LFS.Storage, "/data/gitea/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/data/gitea/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/data/gitea/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/the-archives-dir"},
		{loadActionsFrom, &Actions.LogStorage, "/data/gitea/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/data/gitea/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/data/gitea/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverrideEmpty(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = /data/gitea

[repo-archive]
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/data/gitea/attachments"},
		{loadLFSFrom, &LFS.Storage, "/data/gitea/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/data/gitea/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/data/gitea/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/repo-archive"},
		{loadActionsFrom, &Actions.LogStorage, "/data/gitea/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/data/gitea/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/data/gitea/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalRelativePathOverride(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage]
STORAGE_TYPE = local
PATH = /data/gitea

[repo-archive]
PATH = the-archives-dir
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/data/gitea/attachments"},
		{loadLFSFrom, &LFS.Storage, "/data/gitea/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/data/gitea/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/data/gitea/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/the-archives-dir"},
		{loadActionsFrom, &Actions.LogStorage, "/data/gitea/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/data/gitea/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/data/gitea/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride3(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
STORAGE_TYPE = local
PATH = /data/gitea/archives
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/archives"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride3_5(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
STORAGE_TYPE = local
PATH = a-relative-path
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/a-relative-path"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride4(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
STORAGE_TYPE = local
PATH = /data/gitea/archives

[repo-archive]
PATH = /tmp/gitea/archives
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/tmp/gitea/archives"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride5(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
STORAGE_TYPE = local
PATH = /data/gitea/archives

[repo-archive]
`, []testLocalStoragePathCase{
		{loadAttachmentFrom, &Attachment.Storage, "/appdata/attachments"},
		{loadLFSFrom, &LFS.Storage, "/appdata/lfs"},
		{loadActionsFrom, &Actions.ArtifactStorage, "/appdata/actions_artifacts"},
		{loadPackagesFrom, &Packages.Storage, "/appdata/packages"},
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/data/gitea/archives"},
		{loadActionsFrom, &Actions.LogStorage, "/appdata/actions_log"},
		{loadAvatarsFrom, &Avatar.Storage, "/appdata/avatars"},
		{loadRepoAvatarFrom, &RepoAvatar.Storage, "/appdata/repo-avatars"},
	})
}

func Test_getStorageInheritStorageTypeLocalPathOverride72(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[repo-archive]
STORAGE_TYPE = local
PATH = archives
`, []testLocalStoragePathCase{
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/archives"},
	})
}

func Test_getStorageConfiguration20(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[repo-archive]
STORAGE_TYPE = my_storage
PATH = archives
`)
	require.NoError(t, err)

	require.Error(t, loadRepoArchiveFrom(cfg))
}

func Test_getStorageConfiguration21(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
`, []testLocalStoragePathCase{
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/repo-archive"},
	})
}

func Test_getStorageConfiguration22(t *testing.T) {
	testLocalStoragePath(t, "/appdata", `
[storage.repo-archive]
PATH = archives
`, []testLocalStoragePathCase{
		{loadRepoArchiveFrom, &RepoArchive.Storage, "/appdata/archives"},
	})
}

func Test_getStorageConfiguration23(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[repo-archive]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
`)
	require.NoError(t, err)

	_, err = getStorage(cfg, "", "", nil)
	require.Error(t, err)

	require.NoError(t, loadRepoArchiveFrom(cfg))
	cp := RepoArchive.Storage.ToShadowCopy()
	assert.Equal(t, "******", cp.MinioConfig.AccessKeyID)
	assert.Equal(t, "******", cp.MinioConfig.SecretAccessKey)
}

func Test_getStorageConfiguration24(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[repo-archive]
STORAGE_TYPE = my_archive

[storage.my_archive]
; unsupported, storage type should be defined explicitly
PATH = archives
`)
	require.NoError(t, err)
	require.Error(t, loadRepoArchiveFrom(cfg))
}

func Test_getStorageConfiguration25(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[repo-archive]
STORAGE_TYPE = my_archive

[storage.my_archive]
; unsupported, storage type should be known type
STORAGE_TYPE = unknown // should be local or minio
PATH = archives
`)
	require.NoError(t, err)
	require.Error(t, loadRepoArchiveFrom(cfg))
}

func Test_getStorageConfiguration26(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[repo-archive]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
; wrong configuration
MINIO_USE_SSL = abc
`)
	require.NoError(t, err)
	// require.Error(t, loadRepoArchiveFrom(cfg))
	// FIXME: this should return error but now ini package's MapTo() doesn't check type
	require.NoError(t, loadRepoArchiveFrom(cfg))
}

func Test_getStorageConfiguration27(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[storage.repo-archive]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
MINIO_USE_SSL = true
`)
	require.NoError(t, err)
	require.NoError(t, loadRepoArchiveFrom(cfg))
	assert.Equal(t, "my_access_key", RepoArchive.Storage.MinioConfig.AccessKeyID)
	assert.Equal(t, "my_secret_key", RepoArchive.Storage.MinioConfig.SecretAccessKey)
	assert.True(t, RepoArchive.Storage.MinioConfig.UseSSL)
	assert.Equal(t, "repo-archive/", RepoArchive.Storage.MinioConfig.BasePath)
}

func Test_getStorageConfiguration28(t *testing.T) {
	cfg, err := NewConfigProviderFromData(`
[storage]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
MINIO_USE_SSL = true
MINIO_BASE_PATH = /prefix
`)
	require.NoError(t, err)
	require.NoError(t, loadRepoArchiveFrom(cfg))
	assert.Equal(t, "my_access_key", RepoArchive.Storage.MinioConfig.AccessKeyID)
	assert.Equal(t, "my_secret_key", RepoArchive.Storage.MinioConfig.SecretAccessKey)
	assert.True(t, RepoArchive.Storage.MinioConfig.UseSSL)
	assert.Equal(t, "/prefix/repo-archive/", RepoArchive.Storage.MinioConfig.BasePath)

	cfg, err = NewConfigProviderFromData(`
[storage]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
MINIO_USE_SSL = true
MINIO_BASE_PATH = /prefix

[lfs]
MINIO_BASE_PATH = /lfs
`)
	require.NoError(t, err)
	require.NoError(t, loadLFSFrom(cfg))
	assert.Equal(t, "my_access_key", LFS.Storage.MinioConfig.AccessKeyID)
	assert.Equal(t, "my_secret_key", LFS.Storage.MinioConfig.SecretAccessKey)
	assert.True(t, LFS.Storage.MinioConfig.UseSSL)
	assert.Equal(t, "/lfs", LFS.Storage.MinioConfig.BasePath)

	cfg, err = NewConfigProviderFromData(`
[storage]
STORAGE_TYPE = minio
MINIO_ACCESS_KEY_ID = my_access_key
MINIO_SECRET_ACCESS_KEY = my_secret_key
MINIO_USE_SSL = true
MINIO_BASE_PATH = /prefix

[storage.lfs]
MINIO_BASE_PATH = /lfs
`)
	require.NoError(t, err)
	require.NoError(t, loadLFSFrom(cfg))
	assert.Equal(t, "my_access_key", LFS.Storage.MinioConfig.AccessKeyID)
	assert.Equal(t, "my_secret_key", LFS.Storage.MinioConfig.SecretAccessKey)
	assert.True(t, LFS.Storage.MinioConfig.UseSSL)
	assert.Equal(t, "/lfs", LFS.Storage.MinioConfig.BasePath)
}
