## task description:
Imagine a health tech startup aiming to revolutionize personal wellness through innovative
technology. Your task is to build an API server that helps users track their sleep patterns and
improve their sleep quality. The service will allow users to log their sleep data and track their
sleep trend.

### Requirements
Design and Implementation: Design an API that provides the following functionalities:
 Log Sleep Data: Allow users to log their sleep start and end times, along with
the quality of sleep.
 View Sleep Logs: Enable users to view their past sleep logs.
Core Functionality: Focus on delivering a seamless and user-friendly experience. Consider what kind of data will be handled, how users will interact with the service, and what endpoints will be necessary.
Documentation: Provide a README file that explains:
    The purpose and features of your service
    Instructions on how to set up and run the server
    Examples of how to interact with your API
    A brief explanation of key design decisions
Code Quality: Ensure your code is clean, well-organized, tested, and adheres to best
practices in Go programming.

### 1st iteration idea (unorganized) MoSCoW (Must have, Should have, Could have, Won't have))

1. must have
user's initial data - age (birth date), weight, height, sex, activity level

continous data to store:
 - sleep start/end (logs)
 - users sleep score 1-10 (quality)

GOAL IS: Improve their sleep quality, so storing data is one thing, but lets leverage LLMs, ML models so this project can detect anomalies, problems and give valid and medically relevant output/responses/hints.
Also i guess that's the real value here, cuz we can use speadsheet or notebook to store these logs.

NOTE: in real-life scenario people use smartwatches, smartphones, smartrings, so it would be nice to create generic API that can be used for huge continous data-storage as well as simple start-end sleep logs 

questions to answer once anomalies happen
 1. alcohol?
 2. sickness?
 etc ill read some papers about that 

follow rfc 9457 HTTP error handling https://www.rfc-editor.org/rfc/rfc9457.html

anomalies detection (maybe charts -> but that's UI, so not a part of this task)
 - simple MA

NAPS

2. should have
ML/LLM features, anomalies, suggestions - HUGE TODO

3. could have
evaluations, metrics

4. won't have
pricing, user managment, token validation etc