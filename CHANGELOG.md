# Changelog

## [0.3.0](https://github.com/atani/mysh/compare/v0.2.0...v0.3.0) (2026-03-16)


### Features

* add JSON output format for run and tables commands ([#29](https://github.com/atani/mysh/issues/29)) ([a8fa77f](https://github.com/atani/mysh/commit/a8fa77ff975362ec7d9e8008c4908c17592e08f0))

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
