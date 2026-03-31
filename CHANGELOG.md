# Changelog

## [0.4.0](https://github.com/atani/mysh/compare/v0.3.3...v0.4.0) (2026-03-31)


### Features

* add import command for DBeaver and Sequel Ace connections ([#40](https://github.com/atani/mysh/issues/40)) ([732eff1](https://github.com/atani/mysh/commit/732eff13650d27dbb9cbd914752cafd37ed8ff9c))
* add MySQL Workbench import and cross-platform DBeaver paths ([#42](https://github.com/atani/mysh/issues/42)) ([e051ea3](https://github.com/atani/mysh/commit/e051ea39441340a72253e4bbdf70fba8e913a425))

## [0.3.3](https://github.com/atani/mysh/compare/v0.3.2...v0.3.3) (2026-03-31)


### Bug Fixes

* enable interactive SSH authentication for tunnel connections ([#38](https://github.com/atani/mysh/issues/38)) ([dd2d929](https://github.com/atani/mysh/commit/dd2d929c5c349a917300b2c1e47f059902e94184))

## [0.3.2](https://github.com/atani/mysh/compare/v0.3.1...v0.3.2) (2026-03-27)


### Bug Fixes

* full codebase review findings ([#36](https://github.com/atani/mysh/issues/36)) ([3b7ddae](https://github.com/atani/mysh/commit/3b7ddae1461230751eca6c649c38d164df471dfe))

## [0.3.1](https://github.com/atani/mysh/compare/v0.3.0...v0.3.1) (2026-03-22)


### Bug Fixes

* use platform-specific binary name in Homebrew formula template ([#31](https://github.com/atani/mysh/issues/31)) ([51b10eb](https://github.com/atani/mysh/commit/51b10eb8b2a79047f5b9c14fc509530f8d32e440))

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
