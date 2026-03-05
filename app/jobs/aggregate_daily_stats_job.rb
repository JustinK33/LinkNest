class AggregateDailyStatsJob < ApplicationJob
  queue_as :default

  def perform(date = nil)
    date = (date || 1.day.ago).to_date

    # Get all users with clicks on this date
    user_ids = LinkClick.where(created_at: date.beginning_of_day..date.end_of_day)
                        .distinct
                        .pluck(:user_id)

    user_ids.each do |user_id|
      aggregate_user_day(user_id, date)
    end

    # Clean up old raw click data (keep 90 days)
    cleanup_old_clicks
  end

  private

  def aggregate_user_day(user_id, date)
    user = User.find_by(id: user_id)
    return unless user

    # Get all clicks for this user on this date
    clicks = LinkClick.where(user_id: user_id, created_at: date.beginning_of_day..date.end_of_day)

    total_clicks = clicks.count
    unique_visitors = clicks.distinct.count(:ip_address)

    # Find top link for the day
    top_link_data = clicks.group(:link_id)
                          .select("link_id, COUNT(*) as click_count")
                          .order("click_count DESC")
                          .limit(1)
                          .first

    top_link_id = top_link_data&.link_id
    top_link_clicks = top_link_data&.click_count.to_i || 0

    # Upsert daily stat - MySQL compatible version
    existing = DailyUserStat.find_by(user_id: user_id, date: date)
    if existing
      existing.update(
        total_clicks: total_clicks,
        unique_visitors: unique_visitors,
        top_link_id: top_link_id,
        top_link_clicks: top_link_clicks
      )
    else
      DailyUserStat.create!(
        user_id: user_id,
        date: date,
        total_clicks: total_clicks,
        unique_visitors: unique_visitors,
        top_link_id: top_link_id,
        top_link_clicks: top_link_clicks
      )
    end
  end

  def cleanup_old_clicks
    # Archive clicks older than 90 days
    cutoff_date = 90.days.ago
    LinkClick.where("created_at < ?", cutoff_date).delete_all
  end
end
