# post request
output "http_request_post" {
  value = provider::slack::http_request(
    "https://jsonplaceholder.typicode.com/posts",
    "POST",
    jsonencode({
      "title" : "foo",
      "body" : "bar",
      "userId" : 1
    }),
    {
      "Content-Type" : "application/json"
    }
  )
}

# get request
output "http_request_get" {
  value = provider::slack::http_request(
    "http://httpstat.us", "GET", "", {}
  )
}

# get request returning status code
output "http_request_get_status_code" {
  value = (provider::slack::http_request(
    "http://httpstat.us", "GET", "", {}
  )).status_code
}
