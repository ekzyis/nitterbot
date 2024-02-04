# nitterbot

This is a [Stacker News bot](https://stacker.news/nitter) that fetches every minute all recent posts.

If a post is a twitter link, it adds a comment with nitter links and stores the comment + item id in a sqlite3 database.

If the post already has a comment from the bot (by checking the database), it does not post a comment to prevent duplicates.
