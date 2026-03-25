class Session < ApplicationRecord
  belongs_to :user

  validates :ip_address, presence: true, format: {
    with: /\A(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)|(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\z/,
    message: "must be a valid IPv4 or IPv6 address"
  }
  validates :user_agent, presence: true, length: { maximum: 500 }

  # Session expires after 30 days of inactivity
  EXPIRY_TIME = 30.days

  scope :active, -> { where("updated_at > ?", EXPIRY_TIME.ago) }
  scope :expired, -> { where("updated_at <= ?", EXPIRY_TIME.ago) }

  # Limit sessions per user to prevent session bloat
  MAX_SESSIONS_PER_USER = 10

  before_create :limit_sessions_per_user
  after_create :cleanup_expired_sessions

  def active?
    updated_at > EXPIRY_TIME.ago
  end

  def expired?
    !active?
  end

  def self.cleanup_expired!
    expired.delete_all
  end

  # Check if session might be compromised (different IP/user agent)
  def suspicious_activity?(current_ip, current_user_agent)
    return false unless ip_address.present? && user_agent.present?

    ip_address != current_ip || user_agent != current_user_agent
  end

  private

  def limit_sessions_per_user
    return unless user&.persisted?

    # Remove oldest sessions if user has too many
    excess_count = user.sessions.count - MAX_SESSIONS_PER_USER + 1
    if excess_count > 0
      user.sessions.order(:updated_at).limit(excess_count).delete_all
    end
  end

  def cleanup_expired_sessions
    # Clean up expired sessions periodically (not on every create to avoid performance issues)
    return unless rand < 0.1 # 10% chance to run cleanup

    self.class.cleanup_expired!
  end
end
