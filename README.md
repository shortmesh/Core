# ShortMesh core

## Notes
### Postgres issues
- [Cannot connect to Postgres database](https://github.com/matrix-org/synapse/issues/2780)

### Configuration
- [Setup](https://willlewis.co.uk/blog/posts/stronger-matrix-auth-mas-synapse-docker-compose/)

### Snaypse
```nginx
server {
    listen 443 ssl http2;
    server_name matrix.example.com;

    ssl_certificate /etc/letsencrypt/live/matrix.sherlockwisdom.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/matrix.sherlockwisdom.com/privkey.pem;

    client_max_body_size 50M;

    # MAS-backed client auth routes
    location ~ ^/_matrix/client/(v3|v1)/(login|logout|refresh|auth_metadata|capabilities) {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Synapse endpoints
    location ~ ^(/_matrix|/_synapse/client|/_synapse/mas) {
        proxy_pass http://127.0.0.1:8008;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # .well-known
    location /.well-known/matrix/ {
        alias /var/www/matrix/.well-known/matrix/;
        default_type application/json;
        add_header Access-Control-Allow-Origin *;
    }
}
```

***.well-known/client***
```json
{
  "m.homeserver": {
    "base_url": "https://matrix.sherlockwisdom.com"
  },
  "org.matrix.msc2965.authentication": {
    "issuer": "https://auth.sherlockwisdom.com/",
    "account": "https://auth.sherlockwisdom.com/account/"
  }
}
```
