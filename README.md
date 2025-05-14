# Gator
## What It Is
Gator is an RSS aggregator CLI that allows you (and other users on the same machine) to add, follow, unfollow, and browse RSS feeds.
## Why It Exists
This is a guided project for Boot.dev, and as a result the testing, error handling, and overall functionality of Gator is quite minimal. I probably won't be making any updates to this project for a long time, but maybe I'll come back to it and add a few things for fun at some point.
## Instructions
### Prerequisites
Before you can actually use Gator, you'll need to make sure you have some things set up first. Here are some steps:
- **Install Go**  
  Gator is a Go program, which means you'll need to install Go in order to get it working. Check out the official Go website [here](https://go.dev/).
  
- **Install PostgreSQL**  
  Gator runs on a PostgreSQL database, specifically version 15 or higher. You can find the PostgreSQL website [here](https://www.postgresql.org/).
  
- **Create the PostgreSQL database**  
  You'll need to create a new database for the program to interact with. Try running `psql postgres` in your command line, then running the command `CREATE DATABASE gator;`.
  If that doesn't work, you may want to take a look at the Tutorial section of the [PostgreSQL docs](https://www.postgresql.org/docs/) for your version.
  Regardless of which method you use, I recommend you keep the name of the database as `gator` for simplicity.  
  
- **Set up the database schema**  
  I used [goose](https://github.com/pressly/goose) for this part: `go install github.com/pressly/goose/v3/cmd/goose@latest`  
  The `sql/schema` directory has all the migration files you need. You can just download all the files and stick them in a folder, and then open your command line in that folder.
  Once you've done that, go ahead and run this command there: `goose postgres "postgres://postgres:postgres@localhost:5432/gator" up`
  You'll know it worked if you see something like `2025/05/14 17:29:05 goose: successfully migrated database to version: 5` at the bottom of the output.  
  
- **Install Gator**  
  Once you have everything up and running, you should be able to run `go install github.com/45uperman/gator` in your command line to install Gator and get started.  
### Usage
First thing's first, you'll need to `register` a user for yourself with the program like so: `gator register "your_username_here"`  
You can `register` multiple users and switch between them with the `login` command: `gator login "your_username_here"`  
Registering a user will automatically log you in as that user.  

If you forget your username, try: `gator users`  
  
Now, time for some feeds. In order to add an RSS feed, you'll need to use the `addfeed` command. Here's an example: `gator addfeed "Boot.dev Blog" "https://blog.boot.dev/index.xml"`  
You can `follow` and `unfollow` feeds with the respective commands: `gator follow "https://blog.boot.dev/index.xml"`, `gator unfollow "https://blog.boot.dev/index.xml"` (sorry Lane)  
Adding a feed also follows it.  

But following a feed just means marking it's contents to be fetched when you run the `agg` command: `gator agg 1m`  
This will tell Gator to start fetching the followed feeds in the background (most recently fetched last).  
The `1m` tells Gator how long to wait after checking each feed.  

When you want to actually `browse` your feeds, you can use the command: `gator browse 10`  
The `10` tells Gator to only show the 10 most recent posts.  

Finally, and quite dangerously, you can delete all the users (and subsequently all the other data) from your database with the `reset` command: `gator reset 51420251734`  
That long string of numbers is just the date and time I'm writing this to make it harder to input by mistake.
