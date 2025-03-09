# Gotify matrix bot

This project provides a  bridge between Gotify push notifications and the Matrix messaging platform. It's a maintained continuation of the original `gotify-matrix-bot` project (<https://github.com/Ondolin/gotify-matrix-bot/>), ensuring ongoing support and compatibility for users who rely on this integration.

## Overview

This application acts as an intermediary, forwarding notifications from a Gotify server directly into a specified Matrix room. This enables users to receive real-time updates from various Gotify-enabled services within their preferred Matrix environment.

It supports both plain text and markdown gotify messages. They are rendered as html when sent to Matrix. An optional media downloader can fetch referenced images.

## Installation

### Standalone app / build from source

1. **Clone the Repository:**

    ```bash
    git clone https://github.com/Ondolin/gotify-matrix-bot.git
    ```

2. **Read the config**: adjust the `/config.yaml` according to your needs. See the section [Configuration](#configuration).
3. **Build the application:**

    ```bash
    go build
    ```

4. **Run the application:**

    ```bash
    ./gotify_matrix_bot
    ```

### Docker

You can use docker to run the bot. You need a mount the /data directory in read-write mode. The data directory needs to contain the `config.yaml`. See the section [Configuration](#configuration) for more details. It will also store state data from the running bot.

Sample docker compose snippet:

```yaml
  gotifymatrixhomelab:
    container_name: gotifymatrixhomelab
    image: ghcr.io/maxberger/gotify-matrix-bot:master
    volumes:
      - /your/path/to/data:/data
    restart: unless-stopped    
```

## Configuration

Use [example.config.yaml](example.config.yaml) as a starting point. Copy it onto your system as `config.yaml` and edit your settings. You will need to set the connection information for gotify and matrix.

### Media Downloader

Most Matrix clients will not download images from remote servers. This is why this bot contains a media downloader, which will grab references images, and upload them to Matrix as embedded media. This allows image references in markdown messages, similar to the way they are displayed in the gotify UI.

Since this functionality could also be leak privacy information, you have to explicitly enable it using the allowedHost feature.

```yaml
downloader:
  allowedHosts:
    # Set this to allow image downloading from gotify messages. The format is regexp.
    # If you really want to allow all hosts, set this to ".*"
    - ".*\\.yourdomain\\.com"
    - ".*\\.trusteddomain\\.com"
```

You can set a list of allowed hosts to download images from. If any of the hosts matches, the image will be downloaded and converted into a Matrix media.

If you just want to enable downloading from all hosts, set this to:

```yaml
downloader:
  allowedHosts:
    - ".*"
```

## Contributing

We welcome contributions from the community! If you'd like to improve this project, here's how you can get involved:

* **Bug Reports:** If you encounter any issues, please open a new issue on the [Issues](https://github.com/maxberger/gotify-matrix-bot/issues) page.
* **Feature Requests:** Have an idea for a new feature? Share it by creating a new issue.
* **Pull Requests:** We gladly accept pull requests with bug fixes, improvements, or new features. Please make sure to follow the existing code style and provide clear commit messages.

Your contributions are highly appreciated and help to improve this project for everyone.

## Support

If you need help or have any questions, please feel free to create an issue.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
