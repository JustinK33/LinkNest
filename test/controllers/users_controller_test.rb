require "test_helper"

class UsersControllerTest < ActionDispatch::IntegrationTest
  test "new" do
    get new_user_path
    assert_response :success
  end

  test "create signs in and redirects to homepage" do
    assert_difference("User.count", 1) do
      post users_path, params: {
        user: {
          username: "ada",
          first_name: "Ada",
          last_name: "Lovelace",
          email_address: "ada@example.com",
          password: "Passw0rd!",
          password_confirmation: "Passw0rd!"
        }
      }
    end

    assert_redirected_to root_path
    assert cookies[:session_id]
  end

  test "create renders errors with invalid password" do
    assert_no_difference("User.count") do
      post users_path, params: {
        user: {
          username: "ada2",
          first_name: "Ada",
          last_name: "Lovelace",
          email_address: "ada@example.com",
          password: "password",
          password_confirmation: "password"
        }
      }
    end

    assert_response :unprocessable_entity
  end
end
