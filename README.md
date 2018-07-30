Pasang
======

Single binary to deploy a release via SSH.

# Usage

```
Usage of pasang:
  -dest string
        Deploy destination
  -keep int
        Number of releases to keep (default 5)
  -keysecret string
        Secret for private key
  -local
        Deploy to local folder
  -password string
        Password for SSH authentication
  -privkey string
        Path to private key for SSH authentication
  -src string
        Deploy source
```

`pasang` copy whatever in `src` (file or folder) to a release target at `dest/releases/CURRENT_TIMESTAMP`. If `src` is a file, the release target will be the same file as `src` and if `src` is a folder, the release target will be folder the same as `src`. The number of releases which will be kept can be specified with `keep`. If there are more releases than `keep`, the oldest releases will be deleted so that there will always be `keep` number of releases after `pasang` exits.

`dest` value must be in the format of `user@host:port:/path`, with port is optional. There are 2 supported SSH authentication method, using password or public key. If both are specified, public key authentication takes precedence.

If `local` flag is present, this tool deploys `src` to `dest` in the same machine. In this case `dest` must be a normal `path`.

# Limitation

1. SSH host key check is ignored.
1. Complex path in `dest`, eg: `user@host:port:~/deploy`, may not be supported. Please use absolute path for now.

# Licence

MIT Licence
