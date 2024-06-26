# accounts-api

Please see [the technical documentation](https://docs.dimo.zone/docs) for background.

## Running the app

Make a copy of the settings file and edit as you see fit:

```sh
cp settings.sample.yaml settings.yaml
```

If you kept the default database and SMTP server settings then you'll want to start up the respective containers:

```sh
docker-compose up -d
```

This runs Postgres on port 5432; and [Mailhog](https://github.com/mailhog/MailHog) on port 1025 with a [web interface](http://localhost:8025) on port 8025. A bare-bones dex instance will run on port 5556.

With a fresh database you'll want to run the migrations:

```sh
go run ./cmd/accounts-api migrate
```

Finally, running the app is simple:

```sh
go run ./cmd/accounts-api
```

## Developer notes

Check linting with

```
golangci-lint run
```

Update OpenAPI documentation with

```
swag init --generalInfo cmd/accounts-api/main.go --generatedTime true
```

Questions:

- clean up swagger
- address TODO comments
- Confirm that update user endpoint can now only add country code (since we have dedicated endpoints for linking a wallet or email)
