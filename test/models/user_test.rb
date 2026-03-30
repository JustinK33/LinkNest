require "test_helper"

class UserTest < ActiveSupport::TestCase
  test "downcases and strips email_address" do
    user = User.new(email_address: " DOWNCASED@EXAMPLE.COM ")
    assert_equal("downcased@example.com", user.email_address)
  end

  test "requires first and last name" do
    user = User.new(username: "testuser", email_address: "name@example.com", password: "Passw0rd!", password_confirmation: "Passw0rd!")

    assert_not user.valid?
    assert_includes user.errors[:first_name], "can't be blank"
    assert_includes user.errors[:last_name], "can't be blank"
  end

  test "requires a complex password" do
    user = User.new(username: "testuser", first_name: "A", last_name: "B", email_address: "secure@example.com", password: "password", password_confirmation: "password")

    assert_not user.valid?
    assert_includes user.errors[:password], "must be at least 8 characters and include 1 number and 1 special character"
  end

  test "updates slug when username changes" do
    user = users(:one)

    user.update!(username: "New Name 2026")

    assert_equal "new-name-2026", user.slug
  end

  test "assigns randomized default profile color on create" do
    user = User.create!(
      username: "color-user-#{SecureRandom.hex(3)}",
      first_name: "Color",
      last_name: "User",
      email_address: "color-#{SecureRandom.hex(4)}@example.com",
      password: "Passw0rd!",
      password_confirmation: "Passw0rd!"
    )

    assert_not_equal "#3b82f6", user.profile_color
    assert_match(/\Ahsl\(\d{1,3},\s*\d{1,3}%,\s*\d{1,3}%\)\z/, user.profile_color)
  end
end
