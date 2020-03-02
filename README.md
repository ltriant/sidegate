# SideGate

Share files with friends :)

# What does it do?

1. Serves a directory, and any child directories below it, over HTTP
2. Files can be downloaded from any of the served directories
3. Files can be uploaded into any of the served directories

# Why?

On several occasions friends of mine have wanted to share large files with me, or vice versa, but we have run into the following issues:

1. My MacBook only has USB-C ports, so I can't get the files via a USB flash drive, or external hard drive, unless one of us has an adapter or hub (which we don't), because the U in USB clearly doesn't actually mean universal
2. The friend's computer isn't also a Mac, so we can't use AirDrop
3. The files can be uploaded to a cloud file hosting provider, and then downloaded, but this seems wasteful of bandwidth when we're on the same local network and sitting a few feet from each other
4. The friend's laptop can't run a simple fileserver (like Python's SimpleHTTPServer), because they don't have enough privileges

All of this is a ridiculous and unfortunate reality, so I wrote this dirt-simple program that I can run on my laptop, and then any of my friends with a web browser can either upload files to or download files from my laptop, without us needing to work around some kind of proprietary tech.

# Why not any of the other projects?

There are other projects out there that perform this function, but, of the ones that I saw, I don't like them because:

1. They're often written in Python with third-party dependencies, and I just want a static binary that's easy to compile with no external dependencies or runtimes necessary, that I can also cross-compile for other platforms and share with less technical friends if they want to use this for themselves
2. Worse still, some projects require Apache or nginx, because they're CGI or PHP scripts
3. They have a ton of extra features that I'm not interested in, e.g. user accounts and groups of users (i.e. a database dependency), in-browser file viewers, and tons of JavaScript and CSS effects

This is a lot more than I thought was necessary.

# Install

If you have the [Go toolchain](https://golang.org/) installed, then the `sidegate` service can be installed by:

    $ go get -u github.com/ltriant/sidegate

At some point I may make it installable by OS package managers to remove this dependency.

# Build

The [Go toolchain](https://golang.org/) is necessary to build.

    $ git clone https://github.com/ltriant/sidegate.git
    $ go build

# Running

After installing or building, it can be run without any parameters:

    $ sidegate
    2020/03/02 09:53:43 Serving local directory /Users/ltriant/projects/github/sidegate
    2020/03/02 09:53:43 Listening on http://10.1.18.33:8000

The logged URL - which points to your IP address on your local network - can be shared with your friends, and then amazing file-sharing experiences may commence :)

By default, files are served from current working directory (i.e. whichever directory you ran the server from), but this can be overridden with the `-destDir` parameter.

Also by default, the server listens on port 8000, but can be overridden by the `-port` parameter.

    $ sidegate -destDir /tmp -port 1234

This was intended to be run on a local network only and _not_ the public-facing internet, so there's no SSL.
