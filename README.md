# TmpChat - A Temporary Web Chat Application

TmpChat is a lightweight, real-time web chat application built with Go and WebSockets, designed to be easily deployed using Caddy web server. It provides a simple and clean interface for users to join chat rooms and communicate in real-time.

## Features

- Real-time messaging using WebSockets
- User presence (shows when users join/leave)
- Online user count
- Responsive design that works on desktop and mobile
- Simple deployment with Caddy
- No database required - all data is stored in memory

## Prerequisites

- Go 1.24 or higher
- Caddy v2

## Deployment

1. Build with xcaddy:
   ```bash
   xcaddy build --with github.com/jxskiss/caddy-tmpchat
   ```

2. Edit your Caddyfile to configure tmpchat:
   ```caddy
   yourdomain.com {
       route /chat {
           tmpchat
       }
   }
   ```

3. Run Caddy:
   ```bash
   caddy run --config Caddyfile
   ```

Open `https://yourdomain.com/chat` to start chatting!

## License

This project is open source and available under the [MIT License](LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
