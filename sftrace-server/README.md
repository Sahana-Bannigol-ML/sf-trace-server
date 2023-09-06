### SfTrace-Server Metrics Collector ###

- **Purpose**
	
    This component extracts the metrics to Scale Trace Server. 

- **Working**

    Metrics are queried using apm-server endpoint `localhost:5066/stats`, parsed/converted to prometheus format and exposed to `localhost:2112/metrics`. Prometheus server scrapes these metrics and feeds to HPA for scaling calculation.

- **Scale Logic**

     APM Server uses internal queue to buffer incoming events. Data is buffered in a memory queue before it is published to the configured output. We are using this internal queue size as sizing metrics. Threshold value to scale is 80% of the max number of events the queue can buffer.

        ```
        desiredReplicas = ceil[currentReplicas * ( currentMetricValue / desiredMetricValue )]
        ```
- **Building Image**

    Run below command from `sftrace-server` dir.

    ```
     docker build -t snappyflowml/sftrace-metrics-collector:latest .
    ```
     
