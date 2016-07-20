# piknik
Copy/paste anything over the network

## Blurb
Ever needed a copy/paste clipboard that works on over the network?

Piknik seamlessly transfers URLs, code snipppets, documents, virtually anything, between (possibly firewalled) hosts.
No SSH needed.

Fill in the clipboard ("copy") on host A with whaver comes in on the standard input:

```bash
$ echo clipboard content | pkc
```

Magically retrieve that content from any other host having Piknik installed:

```bash
$ pkp
clipboard content
```

Boom.

In order to bypass firewalls/NAT gatways and to provide persistence, the clipboard content transits via a staging server.

Oh, and nothing transits without encryption and authentication. And data can be shared between different operating systems. Even Windows is kinda supported.

## Installation

This project is written in Go. So, a Go compiler is required, as well as the following incantation:

```bash
$ go get github.com/jedisct1/piknik
```

## Setup

Piknik requires a bunch of keys. Generate them all with

```bash
$ piknik -genkeys
```

The output of that command is all you need to build a configuration file.

Only copy the section for servers on the staging server. Only copy the section for clients on the clients.

Is a host gonna act both as a staging server and as a client? Ponder on it before copying the "hybrid" section, but it's there, just in case.

The default location for the configuration file is `~/.piknik.toml`. With the exception of Windows, where dot-files are not so common. On that platform, the file is simply called `piknik.toml`.

Sample configuration file for a staging server:
```toml
Listen = "0.0.0.0:8075"	# Edit appropriately
Psk = "bf82bab384697243fbf616d3428477a563e33268f0f2307dd14e7245dd8c995d"
SignPk = "0c41ca9b0a1b5fe4daae789534e72329a93a352a6ad73d6f1d368d8eff37271c"
```

Sample configuration file for clients:
```toml
Connect = "127.0.0.1:8075"	# Edit appropriately
EncryptSk = "2f530eb85e59c1977fce726df9f87345206f2a3d40bf91f9e0e9eeec2c59a3e4"
Psk = "bf82bab384697243fbf616d3428477a563e33268f0f2307dd14e7245dd8c995d"
SignPk = "0c41ca9b0a1b5fe4daae789534e72329a93a352a6ad73d6f1d368d8eff37271c"
SignSk = "cecf1d92052f7ba87da36ac3e4a745b64ade8f9e908e52b4f7cd41235dfe74810c41ca9b0a1b5fe4daae789534e72329a93a352a6ad73d6f1d368d8eff37271c"
```

Do not use these, uh? Get your very own keys with the `piknik -genkeys` command.
And edit the `Connect` and `Listen` properties to reflect the staging server IP and port.

Don't like the default config file location? Use the `-config` switch.

## Usage (staging server)

Run the following command on the staging server (or use `runit`, whatever):

```bash
$ piknik -server
```

## Usage (server)

```bash
$ piknik -copy
```

Copy the standard input to the clipboard.

```bash
$ piknik -paste
```

Retrieve the content of the clipboard and spit it to the standard output.
`-paste` is actually a no-op. This is the default action if `-copy` is not being used.

That's it.

Feed it anything. Text, binary data, whatever. As long as it fits in memory.

## Wait...

Wait. Where are the `pkc` and `pkp` commands describer earlier?

Shell aliases:

```bash
alias pkc='piknik -copy'
alias pkp='piknik -paste'
```

Use your own :)

## Protocol

Common:
```
ct: ChaCha20 ke,n (m)
Hk,s: BLAKE2b(domain="SK", key=k, salt=s, size=32)
len(x): x encoded as a 64-bit little endian unsigned integer
n: random 192-bit nonce
r: random 256-bit nonce
sig: Ed25519
v: 1
```

Copy:
```
-> v || r || H0
H0 := Hk,0(v || r)

<- v || H1
H1 := Hk,1(v || H0)

-> 'S' || H2 || len(n || ct) || s || n || ct
s := sig(n || ct)
H2 := Hk,2(H1 || s)

<- Hk,3(H2)
```

Paste:
```
-> v || r || H0
H0 := Hk,0(v || r)

<- v || H1
H1 := Hk,1(v || H0)

-> 'G' || H2
H2 := Hk,2(H1)

<- Hk,3(H2 || sig) || len(n || ct) || s || n || ct
s := sig(n || ct)
```
