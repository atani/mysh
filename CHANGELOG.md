# Changelog

## [0.3.0](https://github.com/atani/mysh/compare/v0.2.0...v0.3.0) (2026-03-16)


### Features

* Add CLI flags for add command and post-add connection test ([#16](https://github.com/atani/mysh/issues/16)) ([fb1e079](https://github.com/atani/mysh/commit/fb1e079e43eaeb86268b1f0236cf38d05c583c92))
* Add keychain support, edit command, and optional connection name ([#10](https://github.com/atani/mysh/issues/10)) ([2bb6ba2](https://github.com/atani/mysh/commit/2bb6ba2b54ca16c109bc688ec3de9b67b8fd9104))
* Add output format support (plain/markdown/csv/pdf) and terminal demo GIF ([8d04399](https://github.com/atani/mysh/commit/8d0439910b86ea5bbed1d3bcf3ba6308667e98ea))
* Add output format support and terminal demo GIF ([dc2a488](https://github.com/atani/mysh/commit/dc2a48811ff6eb9bc34b8da8006e0dcc338153d5))
* Add slice command to extract records as INSERT statements ([#19](https://github.com/atani/mysh/issues/19)) ([9b4bcca](https://github.com/atani/mysh/commit/9b4bcca0aa3817cd71db4f95f0e9bceb6be89a9e))
* Add version flag (-v, --version) ([#12](https://github.com/atani/mysh/issues/12)) ([a146bc0](https://github.com/atani/mysh/commit/a146bc04591757c475cddc4f0e21a1cc11149b98))
* Always mask output for production connections ([#11](https://github.com/atani/mysh/issues/11)) ([48652cc](https://github.com/atani/mysh/commit/48652cc1c201342247103be7d7037d841aea3bf5))
* Default masking for slice and group list output by environment ([#20](https://github.com/atani/mysh/issues/20)) ([d2b16dd](https://github.com/atani/mysh/commit/d2b16ddf1d68384aece3837f563051d65a432069))
* improve environment selection UX in add command ([#18](https://github.com/atani/mysh/issues/18)) ([a946634](https://github.com/atani/mysh/commit/a9466341bf646adccdc062bea51e522dea3bdb47))
* native ドライバで MySQL 4.x old_password 認証に対応 ([#25](https://github.com/atani/mysh/issues/25)) ([18e7760](https://github.com/atani/mysh/commit/18e77608248fecdb653bdb61667d48270797adb9))
* Require interactive confirmation for --raw on production ([#13](https://github.com/atani/mysh/issues/13)) ([01ee1ba](https://github.com/atani/mysh/commit/01ee1ba3bee31c052bc5969b774433c41a829561))
* Unify mask input and add sensible defaults ([#14](https://github.com/atani/mysh/issues/14)) ([55afe6e](https://github.com/atani/mysh/commit/55afe6ed10ae7dc460913b298bc744908dd0fa66))


### Bug Fixes

* Address review findings for slice command and list output ([#21](https://github.com/atani/mysh/issues/21)) ([fe1493f](https://github.com/atani/mysh/commit/fe1493f664850c7b51812276a71d8054c63e0885))
* Allow adding/removing SSH config via edit command ([#15](https://github.com/atani/mysh/issues/15)) ([f0bb44d](https://github.com/atani/mysh/commit/f0bb44d92a1f2582a3f887b3aeecf1bd7557d14e))
* Handle errcheck errors in toCSV by propagating write errors ([4e5e2df](https://github.com/atani/mysh/commit/4e5e2df18159c276cc6a4bf74239f9580128992a))
* Pass MySQL password via MYSQL_PWD env var instead of -p argument ([#24](https://github.com/atani/mysh/issues/24)) ([952ee95](https://github.com/atani/mysh/commit/952ee95217140943fb7627042a24c41e34b3fc51))
* Remove redundant conn.Mask != nil check in run command ([#23](https://github.com/atani/mysh/issues/23)) ([c10a31d](https://github.com/atani/mysh/commit/c10a31d3aaca4d66b3ff0d667fec213eab778935))
* Slow down demo animation and shorten Scene 1 SQL command ([5764c9c](https://github.com/atani/mysh/commit/5764c9c2bd623c7c5b006804bd8bba8b6a8c812b))
* Validate environment to production/staging/development only ([#17](https://github.com/atani/mysh/issues/17)) ([8597a7c](https://github.com/atani/mysh/commit/8597a7cce7d36565d636a279928d5a29dc1ba401))

## [0.2.0](https://github.com/atani/mysh/compare/v0.1.1...v0.2.0) (2026-03-16)


### Features

* native ドライバで MySQL 4.x old_password 認証に対応 ([#25](https://github.com/atani/mysh/issues/25)) ([18e7760](https://github.com/atani/mysh/commit/18e77608248fecdb653bdb61667d48270797adb9))

## [0.1.1](https://github.com/atani/mysh/compare/v0.1.0...v0.1.1) (2026-03-15)


### Bug Fixes

* Address review findings for slice command and list output ([#21](https://github.com/atani/mysh/issues/21)) ([fe1493f](https://github.com/atani/mysh/commit/fe1493f664850c7b51812276a71d8054c63e0885))
* Pass MySQL password via MYSQL_PWD env var instead of -p argument ([#24](https://github.com/atani/mysh/issues/24)) ([952ee95](https://github.com/atani/mysh/commit/952ee95217140943fb7627042a24c41e34b3fc51))
* Remove redundant conn.Mask != nil check in run command ([#23](https://github.com/atani/mysh/issues/23)) ([c10a31d](https://github.com/atani/mysh/commit/c10a31d3aaca4d66b3ff0d667fec213eab778935))

## [0.1.0](https://github.com/atani/mysh/compare/v0.0.10...v0.1.0) (2026-03-15)


### Features

* Add slice command to extract records as INSERT statements ([#19](https://github.com/atani/mysh/issues/19)) ([9b4bcca](https://github.com/atani/mysh/commit/9b4bcca0aa3817cd71db4f95f0e9bceb6be89a9e))
* Default masking for slice and group list output by environment ([#20](https://github.com/atani/mysh/issues/20)) ([d2b16dd](https://github.com/atani/mysh/commit/d2b16ddf1d68384aece3837f563051d65a432069))
