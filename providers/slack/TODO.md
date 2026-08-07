TODO: 1. User Profile Management
Use Case: Managing a user's profile attributes such as name, title, status, and custom fields.
Relevant API: users.profile.set
Terraform Resource: slack_user_profile
Description: Allows administrators to update or create user profile information, including fields like display name, real name, status, custom profile fields, and more.
Fields:
user_id: Slack User ID
real_name: Real name
display_name: Display name
status_text: Status text
status_emoji: Status emoji
custom_fields: A map of custom profile fields
TODO: 3. User Role Management
Use Case: Assigning and managing user roles within a Slack workspace.
Relevant API: No direct API, but achievable through user profile and team management APIs.
Terraform Resource: slack_user_role
Description: Manage user roles such as workspace admin, owner, guest, etc., by interacting with the team management API. While roles can't be set directly through the Slack API, Terraform could wrap workflows that emulate such functionality.
Fields:
user_id: Slack User ID
role: Role to assign (admin, owner, guest)
TODO: 4. User Creation and Invite
Use Case: Automating the creation or invitation of new users to a workspace.
Relevant API: admin.users.invite
Terraform Resource: slack_user_invite
Description: Send invitations to users to join the Slack workspace. This is particularly useful for automating onboarding.
Fields:
email: Email address of the user
real_name: Real name of the user
channels: A list of channels to automatically add the user to
team_id: The team ID (workspace)
TODO: 5. User Deactivation
Use Case: Automating user deactivation (i.e., disabling users who leave the organization).
Relevant API: admin.users.remove
Terraform Resource: slack_user_deactivation
Description: Deactivate or remove a user from the workspace.
Fields:
user_id: Slack User ID
team_id: The team ID (optional if managing multiple workspaces)
TODO: 6. User Group (Alias) Membership
Use Case: Managing user membership in specific user groups or aliases.
Relevant API: usergroups.users.update
Terraform Resource: slack_usergroup_membership
Description: Manage which users are members of specific user groups or aliases, such as "@devops" or "@marketing".
Fields:
group_id: ID of the user group
users: A list of user IDs to be members of the group
TODO: 7. User DM Channel Creation
Use Case: Automating the creation of Direct Message (DM) channels with a user.
Relevant API: conversations.open
Terraform Resource: slack_user_dm
Description: Create a DM conversation with a user, allowing automation tools to send messages directly.
Fields:
user_id: Slack User ID
is_group: (Optional) Whether the DM includes multiple users (for group DM)
TODO: 8. User Time Zone Management
Use Case: Managing a user's time zone settings.
Relevant API: users.profile.set
Terraform Resource: slack_user_timezone
Description: Set or update a user’s time zone.
Fields:
user_id: Slack User ID
timezone: The time zone to set for the user (e.g., "America/Los_Angeles")
TODO: 9. User Email Management
Use Case: Automating updates to a user’s email address.
Relevant API: users.profile.set
Terraform Resource: slack_user_email
Description: Update a user’s email address within their profile.
Fields:
user_id: Slack User ID
email: The new email address for the user
TODO: 10. User Mute/Unmute
Use Case: Automating the process of muting or unmuting a user in a conversation.
Relevant API: conversations.mute, conversations.unmute
Terraform Resource: slack_user_mute
Description: Mute or unmute a user in a specific conversation.
Fields:
user_id: Slack User ID
conversation_id: Conversation ID (channel or DM)
TODO: 11. User Channel Membership Management
Use Case: Adding or removing users from channels.
Relevant API: conversations.invite, conversations.kick
Terraform Resource: slack_user_channel_membership
Description: Manage which users are members of specific channels. Automatically invite or remove users.
Fields:
user_id: Slack User ID
channel_id: Channel ID to add/remove the user from
action: Invite or Kick
TODO: 12. User Presence Status Management
Use Case: Automating changes to user presence (online, away).
Relevant API: users.setPresence
Terraform Resource: slack_user_presence
Description: Set user presence to "active" or "away" automatically.
Fields:
user_id: Slack User ID
presence: Presence status (active, away)
TODO: 13. User Do Not Disturb (DND) Settings
Use Case: Automating the management of Do Not Disturb settings for users.
Relevant API: dnd.setSnooze, dnd.endSnooze
Terraform Resource: slack_user_dnd
Description: Enable or disable DND mode for a user, including scheduling snooze periods.
Fields:
user_id: Slack User ID
snooze_duration: Duration in minutes for which to enable DND
TODO: 14. User Avatar Management
Use Case: Automating the update of user profile pictures (avatars).
Relevant API: users.setPhoto
Terraform Resource: slack_user_avatar
Description: Update a user’s profile picture.
Fields:
user_id: Slack User ID
avatar_url: URL of the avatar image
