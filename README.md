# BettercopeLK

_Better way to search & download Sinhala subtitles_

> **Disclaimer:** This project is an experiment involving Go programming, deployment methods, and platforms. It offers Sinhala subtitle search and download features, but please be cautious as it may not receive regular updates and is not meant for production use. Its functionality and availability may change without prior notice.

## Online versions

- [https://bettercopelk.navinda.xyz](https://bettercopelk.navinda.xyz)

## Currently supported sites

- [Baiscope.lk](https://baiscopelk.com/)
- [Cineru.lk](https://cineru.lk/)
- [Piratelk.com](https://piratelk.com/)
- [Zoom.lk](https://zoom.lk/)

## Requirements

- Go 1.24.1+

## API Documentation

**Base URL**: `https://bettercopelk.navinda.xyz/api/v1`

**Supported source names:**

- `baiscopelk`
- `cineru`
- `piratelk`
- `zoomlk`

### Get available sources

**Endpoint**: `GET /sources`

- **Description**: Get a list of all available subtitle sources.
- **Method**: GET
- **Response**: JSON containing array of available source names.
- **Example Response**:
  ```json
  {
    "sources": ["baiscopelk", "cineru", "piratelk", "zoomlk"]
  }
  ```

### Search subtitles

**Endpoint**: `GET /search?query=movie_name&sources=source1,source2`

- **Description**: Get a list of subtitles for a given movie name.
- **Method**: GET
- **Parameters**:
  - `query` (required): The movie name to search for
  - `sources` (optional): Comma-separated list of sources to search in
- **Response**: JSON array of subtitle results

### Search subtitles (SSE endpoint)

**Endpoint**: `GET /search/stream?query=movie_name&sources=source1,source2`

- **Description**: Get a list of subtitles for a given movie name. This is a Server-Sent Events (SSE) endpoint.
- **Method**: GET
- **Parameters**:
  - `query` (required): The movie name to search for
  - `sources` (optional): Comma-separated list of sources to search in
- **Response**: Server-Sent Events stream of subtitle results

### Download subtitle

**Endpoint**: `GET /download?url=subtitle_post_url&source=source_name`

- **Description**: Download a subtitle from a given source.
- **Method**: GET
- **Parameters**:
  - `url` (required): The URL of the subtitle post
  - `source` (required): The source name of the subtitle
- **Response Content-Type**: `application/zip`