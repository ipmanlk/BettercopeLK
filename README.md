# BettercopeLK

_Better way to search & download Sinhala subtitles_

> **Disclaimer:** This project is an experiment involving Go programming, deployment methods, and platforms. It offers Sinhala subtitle search and download features, but please be cautious as it may not receive regular updates and is not meant for production use. Its functionality and availability may change without prior notice.

## Online versions

- [https://bettercopelk-io.b4a.run/](https://bettercopelk-io.b4a.run/)

## Currently supported sites

- [Baiscope.lk](https://baiscopelk.com/)
- [Cineru.lk](https://cineru.lk/)
- [Piratelk.com](https://piratelk.com/)

## Requirements

- Go 1.20+

## Usage

1. Clone this repository using Git or download as a zip file.

2. Run `go run main.go` to start the server.

## API Documentation

**URL**: `https://bettercopelk-io.b4a.run/api`

**Supported source names:**
- `baiscopelk`
- `cineru`
- `piratelk`

### Search subtitles (SSE endpoint)
GET `/search?query=movie_name`
- **Description**: Get a list of subtitles for a given movie name. This is a Server-Sent Events (SSE) endpoint.
- **Method**: GET

### Download subtitle
GET `/download?postUrl=subtitle_post_url&source=source_name`
- **Description**: Download a subtitle from a given source.
- **Method**: GET
- **Response Content-Type**: `application/zip`

### Bulk download subtitles
POST `/bulk-download`
- **Description**: Download multiple subtitles as a single zip file.
- **Method**: POST
- **Request Format**: JSON
  ```json
  {
    "data": [
      {
        "postUrl": "subtitle_post_url",
        "source": "source_name"
      },
      {
        "postUrl": "subtitle_post_url",
        "source": "source_name"
      }
    ]
  }
  ```
- **Response Content-Type**: `application/zip`

## Screenshots

<img src="https://i.imgur.com/x5zEO6s.png"/>
