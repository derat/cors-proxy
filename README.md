# cors-proxy

`cors-proxy` is a trivial HTTP proxy written in Go that adds a
`Access-Control-Allow-Origin: *` header to responses to let browsers read
cross-origin responses. See [MDN's CORS documentation] for more info.

[MDN's CORS documentation]: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS

## Usage

```
Usage: cors-proxy [flag]...
Forwards HTTP requests and adds 'Access-Control-Allow-Origin: *'.

  -addr string
        host:port to listen on (default "localhost:8000")
  -fastcgi
        Use FastCGI instead of listening on -addr
  -hosts string
        Comma-separated list of allowed forwarding hosts
  -referrers string
        Comma-separated list of allowed referrer hosts
```

In JavaScript code running on `localhost:4000`, you can then do something like
this (after passing `-hosts example.org` and `-referrers localhost:4000`):

```js
fetch(
  'http://localhost:8000/?url=' + encodeURIComponent('https://example.org/foo'),
  {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'arg1=blah&arg2=oof',
    credentials: 'omit',
    mode: 'cors',
  }
)
  .then((resp) => {
    if (!resp.ok) throw new Error(resp.status);
    return resp.text();
  })
  .then((text) => {
    // Do something with |text|.
  });
```
