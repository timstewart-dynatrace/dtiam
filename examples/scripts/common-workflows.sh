#!/bin/bash
# Common dtiam Workflows Reference
#
# This script contains example commands for common IAM management tasks.
# It is meant as a reference - do not run this script directly.
#
# DISCLAIMER: This tool is provided "as-is" without warranty.

set -e

# ============================================================================
# SETUP AND CONFIGURATION
# ============================================================================

# Set up OAuth2 credentials
setup_credentials() {
    dtiam config set-credentials prod \
        --client-id "$DTIAM_CLIENT_ID" \
        --client-secret "$DTIAM_CLIENT_SECRET"

    dtiam config set-context production \
        --account-uuid "$DTIAM_ACCOUNT_UUID" \
        --credentials-ref prod

    dtiam config use-context production
}

# ============================================================================
# GROUP MANAGEMENT
# ============================================================================

# Create a new team group
create_team_group() {
    local team_name="$1"
    local description="$2"

    dtiam create group \
        --name "$team_name" \
        --description "$description"
}

# Add users to a group
add_users_to_group() {
    local group_name="$1"
    shift
    local users=("$@")

    for user in "${users[@]}"; do
        dtiam group add-member "$group_name" --email "$user"
    done
}

# ============================================================================
# POLICY MANAGEMENT
# ============================================================================

# Create a read-only policy
create_viewer_policy() {
    dtiam create policy \
        --name "viewer-policy" \
        --statement "ALLOW settings:objects:read; ALLOW settings:schemas:read;" \
        --description "Read-only access to settings"
}

# Create a binding between group and policy
create_group_binding() {
    local group_name="$1"
    local policy_name="$2"
    local boundary_name="${3:-}"

    if [ -n "$boundary_name" ]; then
        dtiam create binding \
            --group "$group_name" \
            --policy "$policy_name" \
            --boundary "$boundary_name"
    else
        dtiam create binding \
            --group "$group_name" \
            --policy "$policy_name"
    fi
}

# ============================================================================
# BOUNDARY MANAGEMENT
# ============================================================================

# Create a zone boundary
create_zone_boundary() {
    local name="$1"
    local zones="$2"  # comma-separated
    local description="$3"

    dtiam create boundary \
        --name "$name" \
        --zones "$zones" \
        --description "$description"
}

# ============================================================================
# USER MANAGEMENT
# ============================================================================

# Create a new user and add to groups
create_user_with_groups() {
    local email="$1"
    local first_name="$2"
    local last_name="$3"
    local groups="$4"  # comma-separated

    dtiam user create "$email" \
        --first-name "$first_name" \
        --last-name "$last_name" \
        --groups "$groups"
}

# ============================================================================
# SERVICE USER MANAGEMENT
# ============================================================================

# Create a service user for automation
create_automation_service_user() {
    local name="$1"
    local description="$2"
    local group="$3"

    # Create the service user
    # IMPORTANT: Save the output - it contains the client secret!
    dtiam service-user create \
        --name "$name" \
        --description "$description"

    # Add to group for permissions
    if [ -n "$group" ]; then
        dtiam service-user add-to-group "$name" --group "$group"
    fi
}

# ============================================================================
# REPORTING
# ============================================================================

# List all groups with member counts
list_groups_summary() {
    dtiam get groups -o wide
}

# Export all policies to JSON
export_policies() {
    dtiam get policies -o json > policies.json
}

# List all users in a specific group
list_group_members() {
    local group_name="$1"
    dtiam group members "$group_name"
}

# ============================================================================
# CLEANUP
# ============================================================================

# Delete a group (with confirmation)
delete_group() {
    local group_name="$1"

    # Preview first
    dtiam --dry-run delete group "$group_name"

    echo "Press Enter to confirm deletion or Ctrl+C to cancel"
    read -r

    dtiam delete group "$group_name" --force
}

# ============================================================================
# EXAMPLES
# ============================================================================

# Example: Complete team setup
example_team_setup() {
    # Create the team group
    create_team_group "LOB5 Team" "Line of Business 5 engineering team"

    # Create a zone boundary for the team
    create_zone_boundary "lob5-boundary" "LOB5-Prod,LOB5-Staging" "LOB5 zones only"

    # Create binding with boundary
    create_group_binding "LOB5 Team" "devops-policy" "lob5-boundary"

    # Add team members
    add_users_to_group "LOB5 Team" \
        "alice@example.com" \
        "bob@example.com" \
        "charlie@example.com"

    echo "Team setup complete!"
}

echo "This script is a reference - do not run it directly."
echo "Copy the functions you need into your own scripts."
