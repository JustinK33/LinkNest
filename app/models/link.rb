class Link < ApplicationRecord
  belongs_to :user
  has_many :link_clicks, dependent: :destroy
  has_many :hourly_link_stats, dependent: :destroy

  validates :title, presence: true
  validates :url, presence: true, format: { with: URI::DEFAULT_PARSER.make_regexp, message: "must be a valid URL" }
  validates :user_id, presence: true
  validates :position, presence: true, numericality: { only_integer: true }

  scope :ordered, -> { order(:position) }
  scope :active, -> { where(deleted_at: nil) }

  # Update click count from aggregated stats
  def update_click_count!
    total_clicks = link_clicks.count
    update(click_count: total_clicks)
  end

  # Get clicks for a time period
  def clicks_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time)
  end

  # Get unique visitors for a time period
  def unique_visitors_in_range(start_time, end_time)
    link_clicks.where(created_at: start_time..end_time).distinct.count(:ip_address)
  end
end
