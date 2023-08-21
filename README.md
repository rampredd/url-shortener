# url-shortener
URL shortener and it's supporting APIs developed using golang and Redis DB

# what is this
URL shortener is used to shorten long URL like
https://www.accuweather.com/en/in/hyderabad/202190/weather-forecast/202190 converted to OSX8RdCqBtd

# What should I have to run this
Please see go.mod file. <br>
you need to have redis installed

# How to build
Download this repo into your local machine. <br>
go to directory url-shortener  <br>
run below commands  <br>
go mod init  <br>
go mod tidy  <br>
go build -o url-shortener

# How to run
run application with below command  <br>
./url-shortener

# How to test
curl comamnd to shorten URL, after hitting this, save short link from response  <br>

curl --location 'localhost:8020/shorten-url' \
--header 'Content-Type: application/json' \
--data '{
    "destination": "https://www.accuweather.com/en/in/hyderabad/202190/weather-forecast/202190"
}'

curl comamnd to redirect to long URL using short URL  <br>
curl --location 'localhost:8020/short-url/OSX8RdCqBtd'

curl command to see metrics  <br>
curl --location 'localhost:8020/metrics'
