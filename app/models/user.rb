class User < ApplicationRecord
  has_secure_password
  has_one_attached :avatar
  has_many :sessions, dependent: :destroy
  has_many :links, dependent: :destroy
  has_many :link_clicks, dependent: :destroy
  has_many :hourly_link_stats, dependent: :destroy
  has_many :daily_user_stats, dependent: :destroy

  PASSWORD_FORMAT = /\A(?=.*[0-9])(?=.*[^A-Za-z0-9]).{8,}\z/
  SLUG_FORMAT = /\A[a-z0-9]+(?:-[a-z0-9]+)*\z/i

  normalizes :email_address, with: ->(e) { e.strip.downcase }
  normalizes :email, with: ->(e) { e.present? ? e.strip.downcase : e }
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
  validates :email, format: { with: URI::MailTo::EMAIL_REGEXP }, allow_blank: true, if: -> { self.class.column_names.include?("email") }
  validates :profile_color, format: {
    with: /\A(#[0-9A-Fa-f]{3,6}|hsl\(\d{1,3},\s*\d{1,3}%,\s*\d{1,3}%\)|rgb\(\d{1,3},\s*\d{1,3},\s*\d{1,3}\))\z/,
    message: "must be a valid CSS color (hex, hsl, or rgb)"
  }, allow_blank: true
  validate :avatar_is_valid_image
  validate :avatar_content_validation
  validate :slug_available_for_username, on: :create

  before_validation :generate_slug, on: :create
  before_create :generate_profile_color

  def to_param
    slug
  end

  # Safe method to access email (handles missing column)
  def email
    if self.class.column_names.include?("email")
      read_attribute(:email)
    else
      nil
    end
  end

  # Safe method to access phone_number (handles missing column)
  def phone_number
    if self.class.column_names.include?("phone_number")
      read_attribute(:phone_number)
    else
      nil
    end
  end

  # Safe method to set email (handles missing column)
  def email=(value)
    if self.class.column_names.include?("email")
      write_attribute(:email, value)
    end
  end

  # Safe method to set phone_number (handles missing column)
  def phone_number=(value)
    if self.class.column_names.include?("phone_number")
      write_attribute(:phone_number, value)
    end
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

    def generate_profile_color
      return if profile_color.present?

      # Deterministic color generation based on username
      # Ensures same user always gets the same color, but different users get different colors
      hue = (username.sum % 360)
      saturation = 65
      lightness = 50

      self.profile_color = "hsl(#{hue}, #{saturation}%, #{lightness}%)"
    end

    def slug_available_for_username
      return if username.blank?

      candidate_slug = username.to_s.parameterize
      return if candidate_slug.blank?
      return unless User.exists?(slug: candidate_slug)

      errors.add(:username, "is already taken")
      errors.add(:slug, "has already been taken")
    end

    def avatar_is_valid_image
      return unless avatar.attached?

      # Content type validation (basic check)
      allowed_types = [ "image/png", "image/jpeg", "image/jpg", "image/webp" ]
      unless avatar.content_type.in?(allowed_types)
        errors.add(:avatar, "must be a PNG, JPG, or WEBP image")
        return
      end

      # File size validation
      if avatar.byte_size > 5.megabytes
        errors.add(:avatar, "must be 5MB or smaller")
        return
      end

      # Server-side file content validation - check image magic bytes
      avatar.blob.open do |file|
        header = file.read(8)
        valid_image = case avatar.content_type
        when "image/png"
            header[0..7] == "\x89PNG\r\n\x1A\n"
        when "image/jpeg", "image/jpg"
            header[0..1] == "\xFF\xD8"
        when "image/webp"
            header[0..3] == "RIFF" && file.read(4) == "WEBP"
        else
            false
        end

        unless valid_image
          errors.add(:avatar, "file does not appear to be a valid image")
        end
      end
    rescue => e
      errors.add(:avatar, "could not validate file content")
    end

    def avatar_content_validation
      return unless avatar.attached?

      # Additional security: check for embedded scripts or suspicious content
      avatar.blob.open do |file|
        content = file.read(512) # Read first 512 bytes for analysis

        # Check for script injection attempts
        suspicious_patterns = [
          /<script/i,
          /javascript:/i,
          /on\w+\s*=/i, # onclick, onload, etc.
          /eval\(/i,
          /expression\(/i
        ]

        suspicious_patterns.each do |pattern|
          if content.match?(pattern)
            errors.add(:avatar, "file contains potentially malicious content")
            break
          end
        end
      end
    rescue => e
      # Log error but don't fail validation to avoid DOS attacks
      Rails.logger.warn "Error validating image content: #{e.message}"
    end
end
