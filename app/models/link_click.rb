class LinkClick < ApplicationRecord
  belongs_to :user
  belongs_to :link, counter_cache: :click_count

  validates :user_id, presence: true
  validates :link_id, presence: true

  scope :recent, -> { order(created_at: :desc) }
  scope :for_user, ->(user) { where(user_id: user.id) }
  scope :for_link, ->(link) { where(link_id: link.id) }
  scope :in_range, ->(start_time, end_time) { where(created_at: start_time..end_time) }
end
