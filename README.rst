=====
gobot
=====

gobot is a slack app for daily ops automation.


Configuration
=============

Slack
-----

1. register your app at here (https://api.slack.com/apps?new_app=1)
2. go app menu > Bot User then, cureate bot user
3. go app menu > OAuth & Permission then, Install App to Workspace
4. set app endpoint at Interactive Components > Request URL


Github
------

1. Repository > Settings > Webhooks > Add webhook
2. set Payload URL
3. set Content Type to ``application/json``
4. Enable SSL verirication
5. select events