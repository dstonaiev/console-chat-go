Please start server BEFORE any client!

#Run server

go run cmd/server/server.go -p 17100

#Run client

go run cmd/client/client.go <flags>

Example:
go run cmd/client/client.go  -h localhost:17100 -n Dmytro

All flags are optional and provides default values
-h - server host and port to which you are connecting
-n - client username, no special validation on server, just ensure that user with same name was not joined before
-d - set debug mode in the logging (true/false)
-lf - client log file path (ensure file is accessible otherwise stdout will be used)

#Client menu description
1. Start conversation - select contact from list of available (all joined contacts + groups visible for this user). Start chatting, type :q if you want to leave chatting window.
2. Create group chat - create new group. Application validates that group is non-existent. Current user joined automatically
3. Join group chat - user can join to existent group
4. Leave group chat - leave group, if last user left - group deleted automatically
5. Quit - Leave client znd logout from server
