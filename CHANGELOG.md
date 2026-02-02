# Changelog

All notable changes to this project will be documented in this file.

## v0.0.6

### Improvements
- Upgraded dependencies and runtimes to latest versions (#9)

### Bug Fixes
- Fixed redirect when requesting directories (e.g., `http://www.example.com/dir`) to properly handle MainPageSuffix setting, aligning with GCS static website behavior (#8)

## v0.0.5

### Bug Fixes
- Fixed redirect when requesting directories to match GCS MainPageSuffix behavior (#8)

## v0.0.4

### Improvements
- Added support for multiple platform Docker images (linux/amd64, linux/arm64) for better compatibility with Arm CPUs (#7)

## v0.0.3

### New Features
- Added support for Basic Authentication via `GCS_PROXY_BASIC_AUTH` environment variable (#3)
- Added support for OPTIONS requests and CORS headers (#5)

### Bug Fixes
- Fixed incorrect Content-Length header in responses - corrected attrs.Size usage (#4)
- Fixed HEAD method to return correct headers without response body (#6)

## v0.0.2

### Bug Fixes
- Fixed range request length calculation - was 1 byte too short (#2)
- Added comprehensive tests for range request handling (#2)

## v0.0.1

### New Features
- Added conditional request support for GET requests (#1)
- Implemented 304 Not Modified responses for requests with conditional headers (If-None-Match, If-Modified-Since) (#1)
- Added ETag and Last-Modified header support (#1)
