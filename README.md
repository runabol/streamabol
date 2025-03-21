# Streamabol

Streamabol is a lightweight Golang application that converts video files into HLS (HTTP Live Streaming) streams on the fly. It takes a video URL as input and generates an HLS-compliant manifest (`.m3u8`) along with segmented video streams, making it ideal for real-time video streaming applications.

![Player](screenshot_v2.jpg)

## Features
- Converts video files to HLS streams dynamically
- Supports remote video URLs as input
- Simple HTTP API for generating HLS manifests
- Lightweight and efficient, built with Go
- Signed URLs to prevent tampering

## Quickstart

To quickly get started with Streamabol, you can use the prebuilt Docker image:

```bash
docker run -it --rm --name=streamabol -p 8080:8080 runabol/streamabol
```

Play a sample HLS stream: 

```bash
https://hls-player-demo.vercel.app?src=http://localhost:8080/manifest.m3u8?src=http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ElephantsDream.mp4
```

## URL Signing (Optional)

You can sign Streamabol URLs to secure your video streams, prevent unauthorized access, and ensure URL integrity by preventing tampering. This feature ensures that only clients with valid signed URLs can access the HLS manifests and video segments. When enabled, all incoming requests must include a valid `hmac` query parameter to be processed.

### Enabling HMAC Verification
To enable HMAC verification, set the `SECRET_KEY` environment variable to your desired secret key. If `SECRET_KEY` is not set, HMAC verification will be disabled.

```bash
# Enable HMAC verification with a specific key
export SECRET_KEY="1234"
```

### Signing URLs
When HMAC verification is enabled, you must sign your URLs by generating an HMAC-SHA256 signature and appending it as an `hmac` query parameter. The signature should be calculated using:
- The secret key (same as `SECRET_KEY`)
- The URL path and query parameters (excluding the `hmac` parameter)
- URL-encoded parameters where applicable

#### Signing Example
Here's how to sign a URL using `openssl` and `xxd`:

```bash
# Example URL path with encoded parameter
echo -n "/manifest.m3u8?src=http%3A%2F%2Fcommondatastorage.googleapis.com%2Fgtv-videos-bucket%2Fsample%2FElephantsDream.mp4" | openssl dgst -sha256 -hmac "1234" -binary | xxd -p -c 256
# Output: e1c8b8f7e2a3d4c5b6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9

# Resulting signed URL:
# /manifest.m3u8?src=http%3A%2F%2Fcommondatastorage.googleapis.com%2Fgtv-videos-bucket%2Fsample%2FElephantsDream.mp4&hmac=e1c8b8f7e2a3d4c5b6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9
```

Notes:
- Use `echo -n` to prevent adding a newline
- The input should be the exact path and query string (URL-encoded)
- The full request URL would include your domain (e.g., `http://localhost:8080` + signed path)

### Generating a Random Secret Key
For security, you should use a strong, random secret key. Here's how to generate one:

```bash
# Generate a 32-byte random key (hex encoded)
openssl rand -hex 32
# Example output: 7f9c2ba4e8b9d3f0c1e5a7b6d8f9e0c2a3b4c5d6e7f8a9b0c1d2e3f4a5b6d7e8

# Set it as the environment variable
export SECRET_KEY="7f9c2ba4e8b9d3f0c1e5a7b6d8f9e0c2a3b4c5d6e7f8a9b0c1d2e3f4a5b6d7e8"
```

## Prerequisites
- [FFmpeg](https://ffmpeg.org/download.html) installed on your system (used for video processing)

## Build from source
1. Clone the repository:
   ```bash
   git clone https://github.com/runabol/streamabol.git
   ```
2. Navigate to the project directory:
   ```bash
   cd streamabol
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Build the application:
   ```bash
   go build -o streamabol
   ```

## Environment Variables

| Variable      | Description                                 | Default Value |
|---------------|---------------------------------------------|---------------|
| `BASE_DIR`    | The base directory where streams are stored | `os.TempDir()`  |
| `LOG_FORMAT`  | Possible values are `pretty` and `json`     | `pretty` |
| `LOG_LEVEL`   | Possible values are `debug`, `info`, `warn`, `error`, and `fatal`. | `debug`
| `SECRET_KEY`  | Secret key used for HMAC verification       | None          |

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
