# SideGate

Accept any and all file uploads

# Why?

On many occasions I've had friends who have wanted to send large files to me, but have run into the following issues:

1. My MacBook only has USB-C ports, so I can't accept the files via a flash drive, or external hard drive, unless they magically have a USB-C cable, which they never do
2. The friend's laptop isn't a MacBook, so we can't use AirDrop
3. The files can be uploaded to a cloud file hosting provider, and then downloaded, but this seems wasteful of bandwidth when we're on the same local network
4. The friend's laptop can't run a dead-simple fileserver (like Python's SimpleHTTPServer), because they don't have enough privileges

All of this is a ridiculous and unfortunate reality, so I wrote this dirt-simple program that I can run on my laptop, and then any of my friends with a web browser can just dump files onto my laptop without needing to work around some kind of proprietary tech.

# Build and Run

The [Go runtime](https://golang.org/) is necessary to build. There are no dependencies, as the standard library was more than enough for this simple project.

    $ go build
    $ ./sidegate

Get your local IP address (from `ifconfig` or whatever means you prefer), have the uploader browse to `http://1.2.3.4:8000` (where `1.2.3.4` is your IP address), and upload away.

By default, files are dropped into the current working directory (i.e. whichever directory you ran the server from), but this can be overridden with the `-destDir` parameter.

Also by default, the server listens on port 8000, but can be overridden by the `-port` parameter.

    $ ./sidegate -destDir /tmp -port 1234
