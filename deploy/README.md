## Deploy on Kubernetes

Create a Kubernetes secret to securely store your API key. Replace `your-key` with your actual API key in the command below:
```bash
kubectl -n galah create secret generic llm-api-key --from-literal=api_key=your-key
```

Deploy the app using the Kubernetes deployment manifest. 
```bash
kubectl apply -f https://raw.githubusercontent.com/0x4D31/galah/main/deploy/galah.yml
```

### Notes:

- Port configuration: If you need to change the honeypot ports, update the container ports and the load balancer configuration in the Kubernetes manifest accordingly.
- Image updates: The manifest uses the `infosecb/galah:latest` Docker image. Be aware that this image might not always be the most current version. If you require the latest version of the app or need to modify anything in `config.yaml`, consider building and using your own Docker image.
