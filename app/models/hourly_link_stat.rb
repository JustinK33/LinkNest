class HourlyLinkStat < ApplicationRecord
  belongs_to :user
  belongs_to :link

  validates :user_id, presence: true
  validates :link_id, presence: true
  validates :hour, presence: true, uniqueness: { scope: :link_id }

  scope :for_user, ->(user) { where(user_id: user.id) }
  scope :for_link, ->(link) { where(link_id: link.id) }
  scope :in_range, ->(start_time, end_time) { where(hour: start_time..end_time) }
  scope :recent, -> { order(hour: :desc) }

  # Get stats for a specific hour (rounded down)
  def self.for_hour(datetime)
    hour = datetime.beginning_of_hour
    where(hour: hour)
  end
end
