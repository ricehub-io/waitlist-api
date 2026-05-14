# RiceHub Waitlist API

REST API for the [waitlist](https://github.com/ricehub-io/waitlist) frontend. Built with Gin web framework.

## Requirements

- [Go](https://go.dev/dl/) toolchain installed
- A running PostgreSQL database
- An S3-compatible object storage (MinIO, RustFS, AWS S3, etc.)
- [Make](<https://en.wikipedia.org/wiki/Make_(software)>) tool installed

## Running locally

1. Copy `.env.example` to `.env` and fill in the required values:

   ```sh
   cp .env.example .env
   ```

   `DATABASE_URL`, `S3_BASE_URL`, `S3_MEDIA_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, and `ADMIN_SECRET` are required. See [.env.example](.env.example) for the full list and default values.

2. Make sure PostgreSQL database and your S3-compatible object storage are running and reachable.

3. Start the server:

   ```sh
   make run
   ```

   By default, you can access the API on http://127.0.0.1:3000.

## API documentation

Once the server is running, the Swagger UI is available at [/swagger/index.html](http://127.0.0.1:3000/swagger/index.html).

## Building

Compile the binary to `./build/api`:

```sh
make build
```

## Makefile

The [Makefile](Makefile) provides additional targets for formatting, linting, testing, generating Swagger docs, and installing dev tools. Run `make help` to list them:

```sh
make help
```

## License

Licensed under the GNU General Public License v3.0. See [LICENSE](LICENSE) for details.
