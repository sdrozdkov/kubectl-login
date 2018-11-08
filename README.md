# kubectl-login

This is a kubectl plugin for login via CLI with OpenID Connect provider (e.g. DEX)
> kubectl-login only compatible with kubectl v1.12 or higher.  
But also kubectl-login may used as separate binary.

## Requirements

Your OpenID Connect provider must have this endpoint for kubernetes api client into configuration:

Default callback endponit: `http://localhost:33768/auth/callback`

## Installation

Download and place kubectl-login binary anywhere in your `$PATH` with execute permissions.
For further information, see the offical [plugin documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/).

Or you can compile it by self.

```bash
git clone https://github.com/sdrozdkov/kubectl-login
cd kubectl-login
go build
```

## Usage

Plugin takes OpenID Connect issuer URL from your .kube/config, so it must be placed in your .kube/config.

Use username assigned to your oidc provider:

```bash
kubectl login sdrozdkov-oidc
```

After command executed browser will be opened with redirect to OpenID Connect Provider login page.  
Tokens into your .kube/config will be replaced after succesful authenticate at your provider.

## TODO

* Add creation new user profile into .kube/config based on command line args or something else.