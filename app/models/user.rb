class User < ApplicationRecord
  has_secure_password
  has_many :sessions, dependent: :destroy
  has_many :links, dependent: :destroy
  has_many :link_clicks, dependent: :destroy
  has_many :hourly_link_stats, dependent: :destroy
  has_many :daily_user_stats, dependent: :destroy

  PASSWORD_FORMAT = /\A(?=.*[0-9])(?=.*[^A-Za-z0-9]).{8,}\z/
  SLUG_FORMAT = /\A[a-z0-9]+(?:-[a-z0-9]+)*\z/i

  normalizes :email_address, with: ->(e) { e.strip.downcase }
  normalizes :slug, with: ->(s) { s.to_s.parameterize }

  validates :username, presence: true, uniqueness: { case_sensitive: false }
  validates :first_name, presence: true
  validates :last_name, presence: true
  validates :email_address, presence: true, uniqueness: { case_sensitive: false }
  validates :slug, presence: true, uniqueness: true
  validates :slug, format: {
    with: SLUG_FORMAT,
    message: "can only contain lowercase letters, numbers, and hyphens"
  }, if: :validate_slug_format?
  validates :password, format: {
    with: PASSWORD_FORMAT,
    message: "must be at least 8 characters and include 1 number and 1 special character"
  }, if: :password_present?
  validate :slug_available_for_username, on: :create

  before_validation :generate_slug, on: :create

  def to_param
    slug
  end

  private
    def validate_slug_format?
      new_record? || will_save_change_to_slug?
    end

    def password_present?
      password.present?
    end

    def generate_slug
      return if slug.present? || username.blank?

      self.slug = username.to_s.parameterize
    end

    def slug_available_for_username
      return if username.blank?

      candidate_slug = username.to_s.parameterize
      return if candidate_slug.blank?
      return unless User.exists?(slug: candidate_slug)

      errors.add(:username, "is already taken")
      errors.add(:slug, "has already been taken")
    end
end
