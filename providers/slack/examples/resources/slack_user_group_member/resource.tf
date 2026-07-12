resource "slack_user_group_member" "example" {
  usergroup    = "Group 1"
  default_user = "admin@mail.com"
  users        = ["myemail1@mail.com", "myemail2@mail.com"]
}
