The go app is running in a tmux session. tmux a -t 0 will get you in there, then you can do a CTRL-c to stop the app, 
up arrow to re-start it and CTRL b + d to detach. Then re-start nginx with sudo systemctl restart nginx 
