package templates

import _ "embed"

// StandardProxyTemplate 是标准的 Nginx 反向代理配置模板
// 包含基本的 HTTP/HTTPS 设置，假设 SWAG 已经处理了 SSL 证书部分
// SWAG 的 subdomain.conf 通常包含在 server 块中，或者作为单独的 server 块
// 根据 SWAG 文档，subdomain confs 通常放在 /config/nginx/proxy-confs/
// 并且它们通常包含完整的 server 块定义。

const StandardProxyTemplate = `## Version 2023/05/31
# make sure that your dns has a cname set for {{ .Subdomain }}

server {
    listen 443 ssl;
    listen [::]:443 ssl;

    server_name {{ .Subdomain }}.*;

    include /config/nginx/ssl.conf;

    client_max_body_size 0;

    # enable for ldap auth, fill in ldap.conf in the ldap folder
    #include /config/nginx/ldap.conf;

    # enable for Authelia
    #include /config/nginx/authelia-server.conf;

    location / {
        # enable the next two lines for http auth
        #auth_basic "Restricted";
        #auth_basic_user_file /config/nginx/.htpasswd;

        # enable the next two lines for ldap auth
        #auth_request /auth;
        #error_page 401 =200 /ldaplogin;

        # enable for Authelia
        #include /config/nginx/authelia-location.conf;

        include /config/nginx/proxy.conf;
        include /config/nginx/resolver.conf;
        set $upstream_app {{ .ContainerName }};
        set $upstream_port {{ .ContainerPort }};
        set $upstream_proto {{ .Protocol }};
        proxy_pass $upstream_proto://$upstream_app:$upstream_port;

    }

    # additional config block
    {{ .ExtraConfig }}
}
`
