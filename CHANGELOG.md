# Changelog

All notable changes to this project will be documented in this file.

## v0.0.8

- add query parameters when redirect when request http://www.example.com/m/dir (#13 by @ysylife)

## v0.0.7

### Fixed
- Should handle `iterator.Done` to avoid unexpected 502 error (#12)

## v0.0.6

### Changed
- Upgraded Go dependencies and updated Dockerfile base images (#9)

### Fixed
- Fixed redirect behavior when requesting directory paths without trailing slash to properly redirect to index file, matching GCS static website behavior (#8 by @ysylife)

## v0.0.5

### Changed
- Updated CI/CD workflow to build multi-platform Docker images (linux/amd64, linux/arm64) (#7)

## v0.0.4

### Added
- Added support for OPTIONS method with CORS headers from bucket configuration (#5)
- Added Basic Authentication support via `GCS_PROXY_BASIC_AUTH` environment variable (#3)
- Added automatic configuration from GCS bucket's Website settings (MainPageSuffix, NotFoundPage) (#3)

### Fixed
- Fixed HEAD method to read 1 byte instead of full content when checking object existence (#6)
- Fixed Content-Length header calculation for range requests (#4)

## v0.0.3

### Fixed
- Fixed range request length calculation that was 1 byte too short, added comprehensive tests (#2 by @p1ass)

## v0.0.2

### Added
- Added conditional request support (If-None-Match, If-Modified-Since) (#1)
- Added 304 Not Modified response support (#1)
- Added ETag and Last-Modified headers (#1)
- Added LICENSE and README documentation (#1)

## v0.0.1

Initial release.
