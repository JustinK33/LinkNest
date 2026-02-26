class User < ApplicationRecord
  has_secure_password
  has_many :sessions, dependent: :destroy

  PASSWORD_FORMAT = /\A(?=.*[0-9])(?=.*[^A-Za-z0-9]).{8,}\z/

  normalizes :email_address, with: ->(e) { e.strip.downcase }
  validates :username, presence: true
  validates :first_name, presence: true
  validates :last_name, presence: true
  validates :email_address, presence: true, uniqueness: { case_sensitive: false }
  validates :password, format: {
    with: PASSWORD_FORMAT,
    message: "must be at least 8 characters and include 1 number and 1 special character"
  }, if: :password_present?

  private
    def password_present?
      password.present?
    end
end
