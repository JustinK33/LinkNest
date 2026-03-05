class DailyUserStat < ApplicationRecord
  belongs_to :user
  belongs_to :top_link, class_name: "Link", optional: true

  validates :user_id, presence: true
  validates :date, presence: true, uniqueness: { scope: :user_id }

  scope :for_user, ->(user) { where(user_id: user.id) }
  scope :in_range, ->(start_date, end_date) { where(date: start_date..end_date) }
  scope :recent, -> { order(date: :desc) }

  # Get stats for a specific date
  def self.for_date(date)
    where(date: date.to_date)
  end

  # Get trend data (last N days)
  def self.last_days(days = 30)
    where(date: (days.days.ago.to_date)..Date.today)
      .order(date: :asc)
  end
end
