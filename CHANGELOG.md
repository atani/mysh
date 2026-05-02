# Changelog

## [0.5.2](https://github.com/atani/mysh/compare/v0.5.1...v0.5.2) (2026-05-02)


### Bug Fixes

* **mask:** tolerate and warn on whitespace in mask config entries ([#59](https://github.com/atani/mysh/issues/59)) ([fb10379](https://github.com/atani/mysh/commit/fb1037921b1e0fe250cbe822188fa3b2a0a6ee6a))

## [0.5.1](https://github.com/atani/mysh/compare/v0.5.0...v0.5.1) (2026-04-01)


### Bug Fixes

* address review findings from v0.5.0 multi-agent review ([9a99804](https://github.com/atani/mysh/commit/9a99804b5eeb6e86ee1f87befcc8b925f5b1c4f6))
* address review findings from v0.5.0 release ([9206c37](https://github.com/atani/mysh/commit/9206c37afadb413eb3115c45efda073e375f479c))

## [0.5.0](https://github.com/atani/mysh/compare/v0.4.0...v0.5.0) (2026-04-01)


### Features

* add export command and YAML import for sharing connections ([95d9831](https://github.com/atani/mysh/commit/95d983174036f4bfc78c288a76f81de6ec1e0340))
* add export command and YAML import for sharing connections ([3c0fe6d](https://github.com/atani/mysh/commit/3c0fe6d57eac2e9e8d234a0eaceb7d0c3b7d3346))
* add Redash integration for non-engineer database access ([1edc995](https://github.com/atani/mysh/commit/1edc9954fd358e799fb92fcc7ee39fabb2d58c07))
* add Redash integration for non-engineer database access ([6b9ce90](https://github.com/atani/mysh/commit/6b9ce908e98002735e1d6e81f71e4d30c892f48c))
* add Windows binary builds to release workflow ([b8cf62b](https://github.com/atani/mysh/commit/b8cf62b95b833e08ff4ec2edcc51a8de0ba8af00))
* add Windows binary builds to release workflow ([f5a06d6](https://github.com/atani/mysh/commit/f5a06d684204a8aee3fa559eef41a1af7dd4af8e))
* support MYSH_MASTER_PASSWORD env var for non-interactive use ([93a1604](https://github.com/atani/mysh/commit/93a16045bc9a9a37d9563e0daf1e27467e5f8f84))
* support MYSH_MASTER_PASSWORD environment variable for non-interactive use ([6dc122c](https://github.com/atani/mysh/commit/6dc122c51aa1dd4107f5a0f713f1b96a0c118b35))
* test connection after password input during import ([#43](https://github.com/atani/mysh/issues/43)) ([abbe29b](https://github.com/atani/mysh/commit/abbe29b78ae961711f4a63cae24c18b839f5cc91))


### Bug Fixes

* clear XDG_CONFIG_HOME in tests to prevent config path leakage ([b63625c](https://github.com/atani/mysh/commit/b63625c51d76aaf99ea6f88f653e8cbd209038c5))
* localize hardcoded password retry messages ([#45](https://github.com/atani/mysh/issues/45)) ([3993905](https://github.com/atani/mysh/commit/39939051ea69a3f663ef991fe9b432c7edf5936a))
* lowercase error strings to satisfy staticcheck ST1005 ([89b1e09](https://github.com/atani/mysh/commit/89b1e095fefeba941835bf17700ceaa4b5fa52a4))
* lowercase remaining error strings in redash client ([b849158](https://github.com/atani/mysh/commit/b8491580cc8370953ce21a209bdc8a5374f4b631))
* use platform-appropriate config directory and process check on Windows ([e7349a5](https://github.com/atani/mysh/commit/e7349a58279161add820df892c2e1e82bfb3433e))
* use platform-appropriate config directory and process check on Windows ([122cff3](https://github.com/atani/mysh/commit/122cff3b9fb905d5fe06e072ea8ad0d23a6a8172)), closes [#47](https://github.com/atani/mysh/issues/47)

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
