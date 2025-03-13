# Posterify Backend Go

To remember the beauty of golang after years of typescript I decided to add a backend to my favorite side-project (so far) [posterify](https://posterify.dpalmer.in).

It adds functionality to create a user, create multiple collections, and save images to those collections. With their data obviously persisiting across devices and sessions.

The project uses the default library's http module and mux for the server layer, JWT for authentication, sqlx for the database connection layer (go-migrate for migrations), and a devcontainer just to simplify development and running it locally.

It's heavily based on the [go-realworld](https://github.com/0xdod/go-realworld) repo. With parts being carried over verbatim.

## Next steps

This is still not deployed anywhere. So next step is to deploy the turn the server into a self contained docker image and host it somewhere and make changes to the posterify frontend to actually user the server.