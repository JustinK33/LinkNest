class AggregateHourlyStatsJob < ApplicationJob
  queue_as :default

  def perform
    # Get unique (user_id, link_id, hour) combinations from recent click clicks
    # For the last hour
    one_hour_ago = 1.hour.ago.beginning_of_hour
    current_hour = Time.current.beginning_of_hour

    # Active users with clicks in the last hour
    user_ids = LinkClick.where(created_at: one_hour_ago..current_hour)
                        .distinct
                        .pluck(:user_id)

    user_ids.each do |user_id|
      aggregate_user_hour(user_id, one_hour_ago)
    end

    # Note: We no longer need to manually update link click_count
    # because LinkClick now has counter_cache: :click_count
    # This automatically updates Link.click_count when LinkClick records are created/destroyed
  end

  private

  def aggregate_user_hour(user_id, hour)
    links = Link.where(user_id: user_id).pluck(:id)
    return if links.empty?

    links.each do |link_id|
      clicks = LinkClick.where(
        user_id: user_id,
        link_id: link_id,
        created_at: hour..(hour + 1.hour)
      )

      click_count = clicks.count
      unique_visitors = clicks.distinct.count(:ip_address)

      next if click_count.zero?

      # Upsert hourly stat - MySQL compatible version
      existing = HourlyLinkStat.find_by(link_id: link_id, hour: hour)
      if existing
        existing.update(click_count: click_count, unique_visitors: unique_visitors)
      else
        HourlyLinkStat.create!(
          user_id: user_id,
          link_id: link_id,
          hour: hour,
          click_count: click_count,
          unique_visitors: unique_visitors
        )
      end
    end
  end

  def update_link_click_counts
    # Update link.click_count with total from link_clicks using bulk queries to avoid N+1
    # Get all link click counts in one query
    link_counts = LinkClick.group(:link_id).count

    # Update links that have different click counts than their cached value
    link_counts.each do |link_id, actual_count|
      Link.where(id: link_id)
          .where.not(click_count: actual_count)
          .update_all(click_count: actual_count)
    end

    # Also reset click_count to 0 for links that have no clicks but have a non-zero count
    Link.left_joins(:link_clicks)
        .where(link_clicks: { id: nil })
        .where.not(click_count: 0)
        .update_all(click_count: 0)
  end
end
