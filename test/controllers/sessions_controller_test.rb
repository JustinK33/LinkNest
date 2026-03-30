require "test_helper"
require "omniauth"

class SessionsControllerTest < ActionDispatch::IntegrationTest
  setup do
    @user = User.take
    OmniAuth.config.test_mode = true
    OmniAuth.config.mock_auth[:google_oauth2] = nil
  end

  teardown do
    OmniAuth.config.mock_auth[:google_oauth2] = nil
    OmniAuth.config.test_mode = false
  end

  test "new" do
    get new_session_path
    assert_response :success
  end

  test "create with valid credentials" do
    post session_path, params: { email_address: @user.email_address, password: "password" }

    assert_redirected_to root_path
    assert cookies[:session_id]
    assert cookies[:last_auth_provider]
  end

  test "create with invalid credentials" do
    post session_path, params: { email_address: @user.email_address, password: "wrong" }

    assert_redirected_to new_session_path
    assert_nil cookies[:session_id]
  end

  test "destroy" do
    sign_in_as(User.take)

    delete session_path

    assert_redirected_to new_session_path
    assert_empty cookies[:session_id]
  end

  test "oauth callback creates a new user when email is not found" do
    OmniAuth.config.mock_auth[:google_oauth2] = google_auth_hash(email: "oauth-new@example.com", uid: "new-google-uid")

    assert_difference -> { User.count }, 1 do
      post "/auth/google_oauth2/callback"
    end

    created_user = User.find_by(email_address: "oauth-new@example.com")
    assert_not_nil created_user
    assert_equal "google_oauth2", created_user.oauth_provider
    assert_equal "new-google-uid", created_user.oauth_uid
    assert_redirected_to edit_user_path(created_user)
    assert cookies[:session_id]
    assert cookies[:last_auth_provider]

    follow_redirect!
    assert_response :success
    assert_select ".oauth-setup-banner", 1

    get edit_user_path(created_user)
    assert_response :success
    assert_select ".oauth-setup-banner", 0
  end

  test "oauth callback links existing user by email" do
    original_username = @user.username
    original_first_name = @user.first_name
    original_last_name = @user.last_name

    OmniAuth.config.mock_auth[:google_oauth2] = google_auth_hash(email: @user.email_address, uid: "linked-google-uid")

    assert_no_difference -> { User.count } do
      post "/auth/google_oauth2/callback"
    end

    @user.reload
    assert_equal original_username, @user.username
    assert_equal original_first_name, @user.first_name
    assert_equal original_last_name, @user.last_name
    assert_equal "google_oauth2", @user.oauth_provider
    assert_equal "linked-google-uid", @user.oauth_uid
    assert_redirected_to dashboard_path
    assert cookies[:session_id]
    assert cookies[:last_auth_provider]
  end

  test "oauth callback does not overwrite when existing user is linked to a different google uid" do
    @user.update!(oauth_provider: "google_oauth2", oauth_uid: "existing-google-uid")

    OmniAuth.config.mock_auth[:google_oauth2] = google_auth_hash(email: @user.email_address, uid: "different-google-uid")

    assert_no_difference -> { User.count } do
      post "/auth/google_oauth2/callback"
    end

    @user.reload
    assert_equal "existing-google-uid", @user.oauth_uid
    assert_redirected_to new_session_path
    assert_equal "Could not sign in with Google. Please try again.", flash[:alert]
  end

  test "oauth callback handles missing auth hash" do
    OmniAuth.config.mock_auth[:google_oauth2] = :invalid_credentials
    post "/auth/google_oauth2/callback"

    assert_redirected_to %r{\Ahttp://www\.example\.com/auth/failure\?message=invalid_credentials&strategy=google_oauth2\z}

    follow_redirect!
    assert_redirected_to new_session_path
  end

  private

  def google_auth_hash(email:, uid:)
    OmniAuth::AuthHash.new(
      provider: "google_oauth2",
      uid: uid,
      info: {
        email: email,
        first_name: "Test",
        last_name: "User",
        name: "Test User"
      }
    )
  end
end
