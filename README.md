# Gotify matrix bot

This project provides a  bridge between Gotify push notifications and the Matrix messaging platform. It's a maintained continuation of the original `gotify-matrix-bot` project (<https://github.com/Ondolin/gotify-matrix-bot/>), ensuring ongoing support and compatibility for users who rely on this integration.

## Overview

This application acts as an intermediary, forwarding notifications from a Gotify server directly into a specified Matrix room. This enables users to receive real-time updates from various Gotify-enabled services within their preferred Matrix environment.

## Installation and Configuration

Detailed instructions for installation, configuration, and deployment can be found on the original project's [Wiki](https://github.com/Ondolin/gotify-matrix-bot/wiki).

## Getting Started

1. **Clone the Repository:**

    ```bash
    git clone https://github.com/Ondolin/gotify-matrix-bot.git
    ```

2. **Read the config**: adjust the `/config.yaml` according to your needs. Use the `example.config.yaml` as a starting point.
3. **Build the application:**

    ```bash
    go build
    ```

4. **Run the application:**

    ```bash
    ./gotify_matrix_bot
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
