# swag 参考

## config/nginx/site-confs/default 文件

```nginx
## Version 2021/04/27 - Changelog: https://github.com/linuxserver/docker-swag/commits/master/root/defaults/default

error_page 502 /502.html;

# redirect all traffic to https
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    return 301 https://$host$request_uri;
}

# main server block
server {
    listen 443 ssl http2 default_server;
    listen [::]:443 ssl http2 default_server;

    root /config/www;
    index index.html index.htm index.php;

    server_name _;

    # enable subfolder method reverse proxy confs
    include /config/nginx/proxy-confs/*.subfolder.conf;
    
    # all ssl related config moved to ssl.conf
    include /config/nginx/ssl.conf;

    # enable for ldap auth
    #include /config/nginx/ldap.conf;

    # enable for Authelia
    #include /config/nginx/authelia-server.conf;

    client_max_body_size 0;

    location / {
        # enable the next two lines for http auth
        #auth_basic "Restricted";
        #auth_basic_user_file /config/nginx/.htpasswd;

        # enable the next two lines for ldap auth
        #auth_request /auth;
        #error_page 401 =200 /ldaplogin;

        # enable for Authelia
        #include /config/nginx/authelia-location.conf;

        try_files $uri $uri/ /index.html /index.php?$args =404;
    }

    location ~ \.php$ {
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        fastcgi_pass 127.0.0.1:9000;
        fastcgi_index index.php;
        include /etc/nginx/fastcgi_params;
    }

    
# sample reverse proxy config for password protected couchpotato running at IP 192.168.1.50 port 5050 with base url "cp"
# notice this is within the same server block as the base
# don't forget to generate the .htpasswd file as described on docker hub
#    location ^~ /cp {
#        auth_basic "Restricted";
#        auth_basic_user_file /config/nginx/.htpasswd;
#        include /config/nginx/proxy.conf;
#        proxy_pass http://192.168.1.50:5050/cp;
#    }

}

# sample reverse proxy config without url base, but as a subdomain "cp", ip and port same as above
# notice this is a new server block, you need a new server block for each subdomain
#server {
#    listen 443 ssl http2;
#    listen [::]:443 ssl http2;
#
#    root /config/www;
#    index index.html index.htm index.php;
#
#    server_name cp.*;
#
#    include /config/nginx/ssl.conf;
#
#    client_max_body_size 0;
#
#    location / {
#        auth_basic "Restricted";
#        auth_basic_user_file /config/nginx/.htpasswd;
#        include /config/nginx/proxy.conf;
#        proxy_pass http://192.168.1.50:5050;
#    }
#}

# sample reverse proxy config for "heimdall" via subdomain, with ldap authentication
# ldap-auth container has to be running and the /config/nginx/ldap.conf file should be filled with ldap info
# notice this is a new server block, you need a new server block for each subdomain
#server {
#    listen 443 ssl http2;
#    listen [::]:443 ssl http2;
#
#    root /config/www;
#    index index.html index.htm index.php;
#
#    server_name heimdall.*;
#
#    include /config/nginx/ssl.conf;
#
#    include /config/nginx/ldap.conf;
#
#    client_max_body_size 0;
#
#    location / {
#        # the next two lines will enable ldap auth along with the included ldap.conf in the server block
#        auth_request /auth;
#        error_page 401 =200 /ldaplogin;
#
#        include /config/nginx/proxy.conf;
#        resolver 127.0.0.11 valid=30s;
#        set $upstream_app heimdall;
#        set $upstream_port 443;
#        set $upstream_proto https;
#        proxy_pass $upstream_proto://$upstream_app:$upstream_port;
#    }
#}

# sample reverse proxy config for "heimdall" via subdomain, with Authelia
# Authelia container has to be running in the same user defined bridge network, with container name "authelia", and wit>
# notice this is a new server block, you need a new server block for each subdomain
#server {
#    listen 443 ssl http2;
#    listen [::]:443 ssl http2;
#
#    root /config/www;
#    index index.html index.htm index.php;
#
#    server_name heimdall.*;
#
#    include /config/nginx/ssl.conf;
#
#    include /config/nginx/authelia-server.conf;
#
#    client_max_body_size 0;
#
#    location / {
#        # the next line will enable Authelia along with the included authelia-server.conf in the server block
#        include /config/nginx/authelia-location.conf;
#
#        include /config/nginx/proxy.conf;
#        resolver 127.0.0.11 valid=30s;
#        set $upstream_app heimdall;
#        set $upstream_port 443;
#        set $upstream_proto https;
#        proxy_pass $upstream_proto://$upstream_app:$upstream_port;
#    }
#}

# enable subdomain method reverse proxy confs
include /config/nginx/proxy-confs/*.subdomain.conf;
# enable proxy cache for auth
proxy_cache_path cache/ keys_zone=auth_cache:10m;
```

## swag 目录结构参考

```
swag
├── compose.yaml
└── config
    ├── crontabs
    │   └── root
    ├── custom-cont-init.d
    ├── custom-services.d
    ├── dns-conf
    │   ├── aliyun.ini
    │   ├── cloudflare.ini
    │   ├── cloudxns.ini
    │   ├── cpanel.ini
    │   ├── desec.ini
    │   ├── digitalocean.ini
    │   ├── directadmin.ini
    │   ├── dnsimple.ini
    │   ├── dnsmadeeasy.ini
    │   ├── dnspod.ini
    │   ├── domeneshop.ini
    │   ├── gandi.ini
    │   ├── gehirn.ini
    │   ├── google.json
    │   ├── he.ini
    │   ├── hetzner.ini
    │   ├── infomaniak.ini
    │   ├── inwx.ini
    │   ├── ionos.ini
    │   ├── linode.ini
    │   ├── luadns.ini
    │   ├── netcup.ini
    │   ├── njalla.ini
    │   ├── nsone.ini
    │   ├── ovh.ini
    │   ├── rfc2136.ini
    │   ├── route53.ini
    │   ├── sakuracloud.ini
    │   ├── transip.ini
    │   └── vultr.ini
    ├── etc
    │   └── letsencrypt
    │       ├── accounts
    │       ├── archive
    │       ├── csr
    │       ├── keys
    │       ├── live
    │       ├── renewal
    │       └── renewal-hooks
    ├── fail2ban
    │   ├── action.d
    │   │   ├── abuseipdb.conf
    │   │   ├── apf.conf
    │   │   ├── badips.conf
    │   │   ├── badips.py
    │   │   ├── blocklist_de.conf
    │   │   ├── bsd-ipfw.conf
    │   │   ├── cloudflare.conf
    │   │   ├── complain.conf
    │   │   ├── dshield.conf
    │   │   ├── dummy.conf
    │   │   ├── firewallcmd-allports.conf
    │   │   ├── firewallcmd-common.conf
    │   │   ├── firewallcmd-ipset.conf
    │   │   ├── firewallcmd-multiport.conf
    │   │   ├── firewallcmd-new.conf
    │   │   ├── firewallcmd-rich-logging.conf
    │   │   ├── firewallcmd-rich-rules.conf
    │   │   ├── helpers-common.conf
    │   │   ├── hostsdeny.conf
    │   │   ├── ipfilter.conf
    │   │   ├── ipfw.conf
    │   │   ├── iptables-allports.conf
    │   │   ├── iptables-common.conf
    │   │   ├── iptables.conf
    │   │   ├── iptables-ipset-proto4.conf
    │   │   ├── iptables-ipset-proto6-allports.conf
    │   │   ├── iptables-ipset-proto6.conf
    │   │   ├── iptables-multiport.conf
    │   │   ├── iptables-multiport-log.conf
    │   │   ├── iptables-new.conf
    │   │   ├── iptables-xt_recent-echo.conf
    │   │   ├── mail-buffered.conf
    │   │   ├── mail.conf
    │   │   ├── mail-whois-common.conf
    │   │   ├── mail-whois.conf
    │   │   ├── mail-whois-lines.conf
    │   │   ├── mynetwatchman.conf
    │   │   ├── netscaler.conf
    │   │   ├── nftables-allports.conf
    │   │   ├── nftables.conf
    │   │   ├── nftables-multiport.conf
    │   │   ├── nginx-block-map.conf
    │   │   ├── npf.conf
    │   │   ├── nsupdate.conf
    │   │   ├── osx-afctl.conf
    │   │   ├── osx-ipfw.conf
    │   │   ├── pf.conf
    │   │   ├── route.conf
    │   │   ├── sendmail-buffered.conf
    │   │   ├── sendmail-common.conf
    │   │   ├── sendmail.conf
    │   │   ├── sendmail-geoip-lines.conf
    │   │   ├── sendmail-whois.conf
    │   │   ├── sendmail-whois-ipjailmatches.conf
    │   │   ├── sendmail-whois-ipmatches.conf
    │   │   ├── sendmail-whois-lines.conf
    │   │   ├── sendmail-whois-matches.conf
    │   │   ├── shorewall.conf
    │   │   ├── shorewall-ipset-proto6.conf
    │   │   ├── smtp.py
    │   │   ├── symbiosis-blacklist-allports.conf
    │   │   ├── ufw.conf
    │   │   └── xarf-login-attack.conf
    │   ├── fail2ban.sqlite3
    │   ├── filter.d
    │   │   ├── 3proxy.conf
    │   │   ├── alpine-sshd.conf
    │   │   ├── alpine-sshd-ddos.conf
    │   │   ├── apache-auth.conf
    │   │   ├── apache-badbots.conf
    │   │   ├── apache-botsearch.conf
    │   │   ├── apache-common.conf
    │   │   ├── apache-fakegooglebot.conf
    │   │   ├── apache-modsecurity.conf
    │   │   ├── apache-nohome.conf
    │   │   ├── apache-noscript.conf
    │   │   ├── apache-overflows.conf
    │   │   ├── apache-pass.conf
    │   │   ├── apache-shellshock.conf
    │   │   ├── assp.conf
    │   │   ├── asterisk.conf
    │   │   ├── bitwarden.conf
    │   │   ├── botsearch-common.conf
    │   │   ├── centreon.conf
    │   │   ├── common.conf
    │   │   ├── counter-strike.conf
    │   │   ├── courier-auth.conf
    │   │   ├── courier-smtp.conf
    │   │   ├── cyrus-imap.conf
    │   │   ├── directadmin.conf
    │   │   ├── domino-smtp.conf
    │   │   ├── dovecot.conf
    │   │   ├── dropbear.conf
    │   │   ├── drupal-auth.conf
    │   │   ├── ejabberd-auth.conf
    │   │   ├── exim-common.conf
    │   │   ├── exim.conf
    │   │   ├── exim-spam.conf
    │   │   ├── freeswitch.conf
    │   │   ├── froxlor-auth.conf
    │   │   ├── gitlab.conf
    │   │   ├── grafana.conf
    │   │   ├── groupoffice.conf
    │   │   ├── gssftpd.conf
    │   │   ├── guacamole.conf
    │   │   ├── haproxy-http-auth.conf
    │   │   ├── horde.conf
    │   │   ├── ignorecommands
    │   │   ├── kerio.conf
    │   │   ├── lighttpd-auth.conf
    │   │   ├── mongodb-auth.conf
    │   │   ├── monit.conf
    │   │   ├── murmur.conf
    │   │   ├── mysqld-auth.conf
    │   │   ├── nagios.conf
    │   │   ├── named-refused.conf
    │   │   ├── nginx-badbots.conf
    │   │   ├── nginx-botsearch.conf
    │   │   ├── nginx-deny.conf
    │   │   ├── nginx-http-auth.conf
    │   │   ├── nginx-limit-req.conf
    │   │   ├── nsd.conf
    │   │   ├── openhab.conf
    │   │   ├── openwebmail.conf
    │   │   ├── oracleims.conf
    │   │   ├── pam-generic.conf
    │   │   ├── perdition.conf
    │   │   ├── phpmyadmin-syslog.conf
    │   │   ├── php-url-fopen.conf
    │   │   ├── portsentry.conf
    │   │   ├── postfix.conf
    │   │   ├── proftpd.conf
    │   │   ├── pure-ftpd.conf
    │   │   ├── qmail.conf
    │   │   ├── recidive.conf
    │   │   ├── roundcube-auth.conf
    │   │   ├── screensharingd.conf
    │   │   ├── selinux-common.conf
    │   │   ├── selinux-ssh.conf
    │   │   ├── sendmail-auth.conf
    │   │   ├── sendmail-reject.conf
    │   │   ├── sieve.conf
    │   │   ├── slapd.conf
    │   │   ├── softethervpn.conf
    │   │   ├── sogo-auth.conf
    │   │   ├── solid-pop3d.conf
    │   │   ├── squid.conf
    │   │   ├── squirrelmail.conf
    │   │   ├── sshd.conf
    │   │   ├── stunnel.conf
    │   │   ├── suhosin.conf
    │   │   ├── tine20.conf
    │   │   ├── traefik-auth.conf
    │   │   ├── uwimap-auth.conf
    │   │   ├── vsftpd.conf
    │   │   ├── webmin-auth.conf
    │   │   ├── wuftpd.conf
    │   │   ├── xinetd-fail.conf
    │   │   ├── znc-adminlog.conf
    │   │   └── zoneminder.conf
    │   └── jail.local
    ├── geoip2db
    ├── keys
    │   ├── cert.crt
    │   ├── cert.key
    │   └── letsencrypt -> ../etc/letsencrypt/live/aiandar.cn
    ├── log
    │   ├── fail2ban
    │   │   ├── fail2ban.log
    │   │   ├── fail2ban.log.1
    │   │   └── fail2ban.log.2.gz
    │   ├── letsencrypt
    │   │   ├── letsencrypt.log
    │   │   ├── letsencrypt.log.1
    │   │   └── letsencrypt.log.2.gz
    │   ├── logrotate.status
    │   ├── nginx
    │   │   ├── access.log
    │   │   ├── access.log.1
    │   │   ├── access.log.2.gz
    │   │   ├── error.log
    │   │   ├── error.log.1
    │   │   └── error.log.2.gz
    │   └── php
    │       ├── error.log
    │       ├── error.log.1
    │       └── error.log.2.gz
    ├── nginx
    │   ├── authelia-location.conf
    │   ├── authelia-server.conf
    │   ├── dhparams.pem
    │   ├── ldap.conf
    │   ├── nginx.conf
    │   ├── proxy.conf
    │   ├── proxy-confs
    │   │   ├── adguard.subdomain.conf.sample
    │   │   ├── adminer.subfolder.conf.sample
    │   │   ├── adminmongo.subdomain.conf.sample
    │   │   ├── airsonic.subdomain.conf.sample
    │   │   ├── airsonic.subfolder.conf.sample
    │   │   ├── appflowy.subdomain.conf
    │   │   ├── archisteamfarm.subdomain.conf.sample
    │   │   ├── aria2-with-webui.subdomain.conf.sample
    │   │   ├── authelia.subdomain.conf.sample
    │   │   ├── bazarr.subdomain.conf.sample
    │   │   ├── bazarr.subfolder.conf.sample
    │   │   ├── beets.subdomain.conf.sample
    │   │   ├── beets.subfolder.conf.sample
    │   │   ├── bitwarden.subdomain.conf.sample
    │   │   ├── bitwarden.subfolder.conf.sample
    │   │   ├── boinc.subdomain.conf.sample
    │   │   ├── boinc.subfolder.conf.sample
    │   │   ├── booksonic.subdomain.conf.sample
    │   │   ├── booksonic.subfolder.conf.sample
    │   │   ├── bookstack.subdomain.conf.sample
    │   │   ├── calibre.subdomain.conf.sample
    │   │   ├── calibre.subfolder.conf.sample
    │   │   ├── calibre-web.subdomain.conf.sample
    │   │   ├── calibre-web.subfolder.conf.sample
    │   │   ├── chevereto.subdomain.conf.sample
    │   │   ├── chronograf.subdomain.conf.sample
    │   │   ├── chronograf.subfolder.conf.sample
    │   │   ├── cloudreve.subdomain.conf
    │   │   ├── code-server.subdomain.conf.sample
    │   │   ├── codimd.subdomain.conf.sample
    │   │   ├── collabora.subdomain.conf.sample
    │   │   ├── commento.subdomain.conf.sample
    │   │   ├── couchpotato.subdomain.conf.sample
    │   │   ├── couchpotato.subfolder.conf.sample
    │   │   ├── crontabui.subfolder.conf.sample
    │   │   ├── dashy.subdomain.conf.sample
    │   │   ├── deluge.subdomain.conf.sample
    │   │   ├── deluge.subfolder.conf.sample
    │   │   ├── demo.subdomain.conf
    │   │   ├── dillinger.subdomain.conf.sample
    │   │   ├── documentserver.subdomain.conf.sample
    │   │   ├── dokuwiki.subdomain.conf.sample
    │   │   ├── dokuwiki.subfolder.conf.sample
    │   │   ├── domoticz.subdomain.conf.sample
    │   │   ├── domoticz.subfolder.conf.sample
    │   │   ├── dozzle.subdomain.conf.sample
    │   │   ├── dozzle.subfolder.conf.sample
    │   │   ├── drone.subdomain.conf.sample
    │   │   ├── duplicati.subdomain.conf.sample
    │   │   ├── duplicati.subfolder.conf.sample
    │   │   ├── embystat.subdomain.conf.sample
    │   │   ├── emby.subdomain.conf.sample
    │   │   ├── emby.subfolder.conf.sample
    │   │   ├── emulatorjs.subdomain.conf.sample
    │   │   ├── filebot.subdomain.conf.sample
    │   │   ├── filebot.subfolder.conf.sample
    │   │   ├── filebrowser.subdomain.conf.sample
    │   │   ├── filebrowser.subfolder.conf.sample
    │   │   ├── flexget.subdomain.conf.sample
    │   │   ├── flexget.subfolder.conf.sample
    │   │   ├── flood.subdomain.conf.sample
    │   │   ├── flood.subfolder.conf.sample
    │   │   ├── foldingathome.subdomain.conf.sample
    │   │   ├── foundryvtt.subdomain.conf.sample
    │   │   ├── freshrss.subdomain.conf.sample
    │   │   ├── freshrss.subfolder.conf.sample
    │   │   ├── gaps.subdomain.conf.sample
    │   │   ├── gaps.subfolder.conf.sample
    │   │   ├── ghost.subdomain.conf.sample
    │   │   ├── ghost.subfolder.conf.sample
    │   │   ├── gitea.subdomain.conf.sample
    │   │   ├── gitea.subfolder.conf.sample
    │   │   ├── glances.subdomain.conf.sample
    │   │   ├── glances.subfolder.conf.sample
    │   │   ├── gotify.subdomain.conf.sample
    │   │   ├── gotify.subfolder.conf.sample
    │   │   ├── grafana.subdomain.conf.sample
    │   │   ├── grafana.subfolder.conf.sample
    │   │   ├── grocy.subdomain.conf.sample
    │   │   ├── guacamole.subdomain.conf.sample
    │   │   ├── guacamole.subfolder.conf.sample
    │   │   ├── hass-configurator.subdomain.conf.sample
    │   │   ├── headphones.subdomain.conf.sample
    │   │   ├── headphones.subfolder.conf.sample
    │   │   ├── healthchecks.subdomain.conf.sample
    │   │   ├── hedgedoc.subdomain.conf.sample
    │   │   ├── heimdall.subdomain.conf.sample
    │   │   ├── heimdall.subfolder.conf.sample
    │   │   ├── homeassistant.subdomain.conf.sample
    │   │   ├── homebridge.subdomain.conf.sample
    │   │   ├── homer.subdomain.conf.sample
    │   │   ├── huginn.subdomain.conf.sample
    │   │   ├── influxdb.subdomain.conf.sample
    │   │   ├── jackett.subdomain.conf.sample
    │   │   ├── jackett.subfolder.conf.sample
    │   │   ├── jdownloader.subdomain.conf.sample
    │   │   ├── jellyfin.subdomain.conf.sample
    │   │   ├── jellyfin.subfolder.conf.sample
    │   │   ├── jenkins.subfolder.conf.sample
    │   │   ├── kanzi.subdomain.conf.sample
    │   │   ├── kanzi.subfolder.conf.sample
    │   │   ├── komga.subdomain.conf.sample
    │   │   ├── komga.subfolder.conf.sample
    │   │   ├── lazylibrarian.subdomain.conf.sample
    │   │   ├── lazylibrarian.subfolder.conf.sample
    │   │   ├── librespeed.subdomain.conf.sample
    │   │   ├── lidarr.subdomain.conf.sample
    │   │   ├── lidarr.subfolder.conf.sample
    │   │   ├── lychee.subdomain.conf.sample
    │   │   ├── mailu.subdomain.conf.sample
    │   │   ├── mailu.subfolder.conf.sample
    │   │   ├── matomo.subdomain.conf.sample
    │   │   ├── mealie.subdomain.conf.sample
    │   │   ├── medusa.subdomain.conf.sample
    │   │   ├── medusa.subfolder.conf.sample
    │   │   ├── metube.subdomain.conf.sample
    │   │   ├── metube.subfolder.conf.sample
    │   │   ├── miniflux.subdomain.conf.sample
    │   │   ├── miniflux.subfolder.conf.sample
    │   │   ├── monitorr.subdomain.conf.sample
    │   │   ├── monitorr.subfolder.conf.sample
    │   │   ├── mstream.subdomain.conf.sample
    │   │   ├── mylar.subdomain.conf.sample
    │   │   ├── mylar.subfolder.conf.sample
    │   │   ├── mytinytodo.subfolder.conf.sample
    │   │   ├── n8n.subdomain.conf.disabled
    │   │   ├── n8n.subdomain.conf.sample
    │   │   ├── navidrome.subdomain.conf.sample
    │   │   ├── netboot.subdomain.conf.sample
    │   │   ├── netdata.subdomain.conf.sample
    │   │   ├── netdata.subfolder.conf.sample
    │   │   ├── nextcloud.subdomain.conf.sample
    │   │   ├── nextcloud.subfolder.conf.sample
    │   │   ├── nzbget.subdomain.conf.sample
    │   │   ├── nzbget.subfolder.conf.sample
    │   │   ├── nzbhydra.subdomain.conf.sample
    │   │   ├── nzbhydra.subfolder.conf.sample
    │   │   ├── octoprint.subdomain.conf.sample
    │   │   ├── ombi.subdomain.conf.sample
    │   │   ├── ombi.subfolder.conf.sample
    │   │   ├── openhab.subdomain.conf.sample
    │   │   ├── openvpn-as.subdomain.conf.sample
    │   │   ├── openvscode-server.subdomain.conf.sample
    │   │   ├── organizr-auth.subfolder.conf.sample
    │   │   ├── organizr.subdomain.conf.sample
    │   │   ├── organizr.subfolder.conf.sample
    │   │   ├── osticket.subdomain.conf.sample
    │   │   ├── overseerr.subdomain.conf.sample
    │   │   ├── papermerge.subdomain.conf.sample
    │   │   ├── petio.subdomain.conf.sample
    │   │   ├── petio.subfolder.conf.sample
    │   │   ├── photoprism.subdomain.conf.sample
    │   │   ├── phpmyadmin.subdomain.conf.sample
    │   │   ├── phpmyadmin.subfolder.conf.sample
    │   │   ├── picard.subfolder.conf.sample
    │   │   ├── pihole.subdomain.conf.sample
    │   │   ├── pihole.subfolder.conf.sample
    │   │   ├── piwigo.subdomain.conf.sample
    │   │   ├── pixelfed.subdomain.conf.sample
    │   │   ├── plex.subdomain.conf.sample
    │   │   ├── plex.subfolder.conf.sample
    │   │   ├── plexwebtools.subdomain.conf.sample
    │   │   ├── plexwebtools.subfolder.conf.sample
    │   │   ├── podgrab.subdomain.conf.sample
    │   │   ├── portainer.subdomain.conf.sample
    │   │   ├── portainer.subfolder.conf.sample
    │   │   ├── privatebin.subdomain.conf.sample
    │   │   ├── prometheus.subdomain.conf.sample
    │   │   ├── prowlarr.subdomain.conf.sample
    │   │   ├── prowlarr.subfolder.conf.sample
    │   │   ├── pydio-cells.subdomain.conf.sample
    │   │   ├── pydio.subdomain.conf.sample
    │   │   ├── pyload.subdomain.conf.sample
    │   │   ├── pyload.subfolder.conf.sample
    │   │   ├── qbittorrent.subdomain.conf.sample
    │   │   ├── qbittorrent.subfolder.conf.sample
    │   │   ├── quassel-web.subdomain.conf.sample
    │   │   ├── quassel-web.subfolder.conf.sample
    │   │   ├── radarr.subdomain.conf.sample
    │   │   ├── radarr.subfolder.conf.sample
    │   │   ├── raneto.subdomain.conf.sample
    │   │   ├── rclone.subfolder.conf.sample
    │   │   ├── readarr.subdomain.conf.sample
    │   │   ├── readarr.subfolder.conf.sample
    │   │   ├── README.md
    │   │   ├── recipes.subdomain.conf.sample
    │   │   ├── requestrr.subdomain.conf.sample
    │   │   ├── resilio-sync.subdomain.conf.sample
    │   │   ├── rutorrent.subdomain.conf.sample
    │   │   ├── rutorrent.subfolder.conf.sample
    │   │   ├── sabnzbd.subdomain.conf.sample
    │   │   ├── sabnzbd.subfolder.conf.sample
    │   │   ├── scope.subfolder.conf.sample
    │   │   ├── scrutiny.subdomain.conf.sample
    │   │   ├── seafile.subdomain.conf.sample
    │   │   ├── service.subdomain.conf
    │   │   ├── sickchill.subdomain.conf.sample
    │   │   ├── sickchill.subfolder.conf.sample
    │   │   ├── sickrage.subdomain.conf.sample
    │   │   ├── sickrage.subfolder.conf.sample
    │   │   ├── skyhook.subdomain.conf.sample
    │   │   ├── smokeping.subdomain.conf.sample
    │   │   ├── smokeping.subfolder.conf.sample
    │   │   ├── sonarr.subdomain.conf.sample
    │   │   ├── sonarr.subfolder.conf.sample
    │   │   ├── statping.subdomain.conf.sample
    │   │   ├── synapse.subdomain.conf.sample
    │   │   ├── synclounge.subdomain.conf.sample
    │   │   ├── synclounge.subfolder.conf.sample
    │   │   ├── syncthing.subdomain.conf.sample
    │   │   ├── syncthing.subfolder.conf.sample
    │   │   ├── taisun.subdomain.conf.sample
    │   │   ├── tasmobackup.subdomain.conf.sample
    │   │   ├── tautulli.subdomain.conf.sample
    │   │   ├── tautulli.subfolder.conf.sample
    │   │   ├── tdarr.subdomain.conf.sample
    │   │   ├── _template.subdomain.conf.sample
    │   │   ├── _template.subfolder.conf.sample
    │   │   ├── thelounge.subdomain.conf.sample
    │   │   ├── thelounge.subfolder.conf.sample
    │   │   ├── transmission.subdomain.conf.sample
    │   │   ├── transmission.subfolder.conf.sample
    │   │   ├── ubooquity.subdomain.conf.sample
    │   │   ├── ubooquity.subfolder.conf.sample
    │   │   ├── unifi-controller.subdomain.conf.sample
    │   │   ├── uptime-kuma.subdomain.conf.sample
    │   │   ├── wallabag.subdomain.conf.sample
    │   │   ├── webtop.subdomain.conf.sample
    │   │   ├── yacht.subdomain.conf.sample
    │   │   ├── youtube-dl-server.subdomain.conf.sample
    │   │   ├── youtube-dl.subfolder.conf.sample
    │   │   ├── znc.subdomain.conf.sample
    │   │   ├── znc.subfolder.conf.sample
    │   │   └── zwavejs2mqtt.subdomain.conf.sample
    │   ├── resolver.conf
    │   ├── site-confs
    │   │   └── default
    │   ├── ssl.conf
    │   └── worker_processes.conf
    ├── php
    │   ├── php-local.ini
    │   └── www2.conf
    └── www
        ├── 502.html
        └── index.html
```