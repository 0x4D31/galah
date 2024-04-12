<img align="left" src="images/galah.png" width="210px">

TL;DR: Galah (/ɡəˈlɑː/ - pronounced ‘guh-laa’) is an LLM (Large Language Model) powered web honeypot, currently compatible with the OpenAI API, that is able to mimic various applications and dynamically respond to arbitrary HTTP requests.

## Description

Named after the clever Australian parrot known for its mimicry, Galah mirrors this trait in its functionality. Unlike traditional web honeypots that rely on a manual and limiting method of emulating numerous web applications or vulnerabilities, Galah adopts a novel approach. This LLM-powered honeypot mimics various web applications by dynamically crafting relevant (and occasionally foolish) responses, including HTTP headers and body content, to arbitrary HTTP requests. Fun fact: in Aussie English, [Galah](https://www.macquariedictionary.com.au/blog/article/728/) also means fool!

I’ve deployed a cache for the LLM-generated responses (the cache duration can be customized in the config file) to avoid generating multiple responses for the same request and to reduce the cost of the OpenAI API. The cache stores responses per port, meaning if you probe a specific port of the honeypot, the generated response won’t be returned for the same request on a different port.

The prompt is the most crucial part of this honeypot! You can update the prompt in the config file, but be sure not to change the part that instructs the LLM to generate the response in the specified JSON format.

> **Note:** Galah was a fun weekend project I created to evaluate the capabilities of LLMs in generating HTTP messages, and it is not intended for production use. The honeypot may be fingerprinted based on its response time, non-standard, or sometimes weird responses, and other network-based techniques. Use this tool at your own risk, and be sure to set usage limits for your OpenAI API.

### Future Enhancements

- Rule-Based Response: The new version of Galah will employ a dynamic, rule-based approach, adding more control over response generation. This will further reduce OpenAI API costs and increase the accuracy of the generated responses.

- Response Database: It will enable you to generate and import a response database. This ensures the honeypot only turns to the OpenAI API for unknown or new requests. I’m also working on cleaning up and sharing my own database.

- Support for Other LLMs.

## Getting Started

### Local deployment

- Ensure you have Go version 1.20+ installed.
- Create an OpenAI API key from [here](https://platform.openai.com/api-keys).
- If you want to serve over HTTPS, generate TLS certificates.
- Clone the repo and install the dependencies.
- Update the `config.yaml` file.
- Build and run the Go binary!

```
% git clone git@github.com:0x4D31/galah.git
% cd galah
% go mod download
% go build  
% ./galah -i en0 -v

 ██████   █████  ██       █████  ██   ██ 
██       ██   ██ ██      ██   ██ ██   ██ 
██   ███ ███████ ██      ███████ ███████ 
██    ██ ██   ██ ██      ██   ██ ██   ██ 
 ██████  ██   ██ ███████ ██   ██ ██   ██ 
  llm-based web honeypot // version 1.0
       author: Adel "0x4D31" Karimi

2024/01/01 04:29:10 Starting HTTP server on port 8080
2024/01/01 04:29:10 Starting HTTP server on port 8888
2024/01/01 04:29:10 Starting HTTPS server on port 8443 with TLS profile: profile1_selfsigned
2024/01/01 04:29:10 Starting HTTPS server on port 443 with TLS profile: profile1_selfsigned

2024/01/01 04:35:57 Received a request for "/.git/config" from [::1]:65434
2024/01/01 04:35:57 Request cache miss for "/.git/config": Not found in cache
2024/01/01 04:35:59 Generated HTTP response: {"Headers": {"Content-Type": "text/plain", "Server": "Apache/2.4.41 (Ubuntu)", "Status": "403 Forbidden"}, "Body": "Forbidden\nYou don't have permission to access this resource."}
2024/01/01 04:35:59 Sending the crafted response to [::1]:65434

^C2024/01/01 04:39:27 Received shutdown signal. Shutting down servers...
2024/01/01 04:39:27 All servers shut down gracefully.
```

### Run in Docker

- Ensure you have Docker CE or EE installed locally
- Create an OpenAI API key from [here](https://platform.openai.com/api-keys).
- Run the docker container

```bash
docker run -d -p 8080:8080 -p 8443:8443 -p 443:443 -p 8080:8080 -e API_KEY='OpenAI_API_Key_Here' infosecb/galah:latest
```

### Deploy in Kubernetes

- Run the Kubernetes deployment manifest files
```bash
kubectl apply -f https://raw.githubusercontent.com/0x4D31/galah/main/deploy/galah.yml
```
- Create the openai-api-key secret with your OpenAI API key (replace `your-key`)
```bash
kubectl -n galah create secret generic openai-api-key --from-literal=api_key=your-key
```

## Example Responses

Here are some example responses:

### Example 1
```
% curl http://localhost:8080/login.php
<!DOCTYPE html><html><head><title>Login Page</title></head><body><form action='/submit.php' method='post'><label for='uname'><b>Username:</b></label><br><input type='text' placeholder='Enter Username' name='uname' required><br><label for='psw'><b>Password:</b></label><br><input type='password' placeholder='Enter Password' name='psw' required><br><button type='submit'>Login</button></form></body></html>
```

JSON log record:
```
{"timestamp":"2024-01-01T05:38:08.854878","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"51978","sensorName":"home-sensor","port":"8080","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/login.php","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Content-Type":"text/html","Server":"Apache/2.4.38"},"body":"\u003c!DOCTYPE html\u003e\u003chtml\u003e\u003chead\u003e\u003ctitle\u003eLogin Page\u003c/title\u003e\u003c/head\u003e\u003cbody\u003e\u003cform action='/submit.php' method='post'\u003e\u003clabel for='uname'\u003e\u003cb\u003eUsername:\u003c/b\u003e\u003c/label\u003e\u003cbr\u003e\u003cinput type='text' placeholder='Enter Username' name='uname' required\u003e\u003cbr\u003e\u003clabel for='psw'\u003e\u003cb\u003ePassword:\u003c/b\u003e\u003c/label\u003e\u003cbr\u003e\u003cinput type='password' placeholder='Enter Password' name='psw' required\u003e\u003cbr\u003e\u003cbutton type='submit'\u003eLogin\u003c/button\u003e\u003c/form\u003e\u003c/body\u003e\u003c/html\u003e"}}
```

### Example 2

```
% curl http://localhost:8080/.aws/credentials
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-west-2
```

JSON log record:
```
{"timestamp":"2024-01-01T05:40:34.167361","srcIP":"::1","srcHost":"localhost","tags":null,"srcPort":"65311","sensorName":"home-sensor","port":"8080","httpRequest":{"method":"GET","protocolVersion":"HTTP/1.1","request":"/.aws/credentials","userAgent":"curl/7.71.1","headers":"User-Agent: [curl/7.71.1], Accept: [*/*]","headersSorted":"Accept,User-Agent","headersSortedSha256":"cf69e186169279bd51769f29d122b07f1f9b7e51bf119c340b66fbd2a1128bc9","body":"","bodySha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},"httpResponse":{"headers":{"Connection":"close","Content-Encoding":"gzip","Content-Length":"126","Content-Type":"text/plain","Server":"Apache/2.4.51 (Unix)"},"body":"[default]\naws_access_key_id = AKIAIOSFODNN7EXAMPLE\naws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\nregion = us-west-2"}}
```

Okay, that was impressive!

### Example 3

Now, let's do some sort of adversarial testing!

```
% curl http://localhost:8888/are-you-a-honeypot
No, I am a server.`
```

JSON log record:
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

You're a galah, mate!
