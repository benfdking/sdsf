# list all users
data "slack_users" "all" {
}

# must be valid email. requires: users:read.email
data "slack_users" "filter_by_email" {
  email = "john@domain.com"
}

# list user(s) based on user name like filter
data "slack_users" "filter_by_name" {
  name = "john"
}

# list user(s) based on user name and email like filters
data "slack_users" "filter_by_name_and_email" {
  email = "john@domain.com"
  name = "john"
}
