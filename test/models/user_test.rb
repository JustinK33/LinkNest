require "test_helper"

class UserTest < ActiveSupport::TestCase
  test "downcases and strips email_address" do
    user = User.new(email_address: " DOWNCASED@EXAMPLE.COM ")
    assert_equal("downcased@example.com", user.email_address)
  end

  test "requires first and last name" do
    user = User.new(email_address: "name@example.com", password: "Passw0rd!", password_confirmation: "Passw0rd!")

    assert_not user.valid?
    assert_includes user.errors[:first_name], "can't be blank"
    assert_includes user.errors[:last_name], "can't be blank"
  end

  test "requires a complex password" do
    user = User.new(first_name: "A", last_name: "B", email_address: "secure@example.com", password: "password", password_confirmation: "password")

    assert_not user.valid?
    assert_includes user.errors[:password], "must be at least 8 characters and include 1 number and 1 special character"
  end
end
