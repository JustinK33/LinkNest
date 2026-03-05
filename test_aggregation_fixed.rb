#!/usr/bin/env ruby

ENV['RAILS_ENV'] ||= 'development'
require_relative 'config/environment'

puts "📈 Testing Aggregation Pipeline (Fixed Timestamps)"
puts "=" * 60

# Clear old data
puts "1️⃣  Clearing test data..."
LinkClick.delete_all
HourlyLinkStat.delete_all
DailyUserStat.delete_all
puts "   ✅ Database cleared"
puts ""

# Create test clicks with timestamps IN THE LAST HOUR
puts "2️⃣  Creating test clicks with proper timestamps..."
now = Time.current
last_hour = now.beginning_of_hour - 1.hour  # Last complete hour
next_hour = last_hour + 1.hour

puts "   Last hour range: #{last_hour.strftime('%H:%M')} - #{next_hour.strftime('%H:%M')}"

5.times do |i|
  LinkClick.create!(
    link_id: 1,
    user_id: 1,
    created_at: last_hour + (i * 10).minutes,  # Spread across last hour
    referrer: "https://google.com",
    user_agent: "Mozilla/5.0",
    ip_address: "192.168.1.#{i}",
    country_code: "US",
    device_type: "desktop",
    browser_name: "Chrome"
  )
end
puts "   ✅ Created 5 test clicks in last hour"
puts ""

# Check total clicks
total_clicks = LinkClick.count
puts "3️⃣  Total LinkClick records: #{total_clicks}"
puts ""

# Run hourly aggregation
puts "4️⃣  Running AggregateHourlyStatsJob..."
puts "   Looking for clicks between:"
puts "   - #{(1.hour.ago.beginning_of_hour).strftime('%Y-%m-%d %H:%M')}"
puts "   - #{Time.current.beginning_of_hour.strftime('%Y-%m-%d %H:%M')}"

AggregateHourlyStatsJob.perform_now
puts "   ✅ Hourly aggregation complete"
puts ""

# Check hourly stats
hourly_count = HourlyLinkStat.count
puts "5️⃣  HourlyLinkStat records created: #{hourly_count}"

if hourly_count > 0
  HourlyLinkStat.all.each do |stat|
    puts "   ✅ Hourly stat:"
    puts "      - Link: #{stat.link.title}"
    puts "      - Hour: #{stat.hour.strftime('%Y-%m-%d %H:00')}"
    puts "      - Clicks: #{stat.click_count}"
    puts "      - Unique visitors: #{stat.unique_visitors}"
  end
else
  puts "   ❌ No hourly stats created!"
  puts "\n   Debugging - LinkClicks in last hour:"
  LinkClick.where(created_at: (1.hour.ago.beginning_of_hour)..Time.current.beginning_of_hour).each do |click|
    puts "      - #{click.created_at.strftime('%H:%M:%S')} | Link #{click.link_id}"
  end
end
puts ""

# Check link cache
puts "6️⃣  Verifying Link.click_count cache..."
link = Link.find(1)
puts "   ✅ Link click_count: #{link.click_count}"
puts ""

# Test daily aggregation
puts "7️⃣  Running AggregateDailyStatsJob..."
AggregateDailyStatsJob.perform_now(Date.current)  # Pass today's date
puts "   ✅ Daily aggregation complete"
puts ""

# Check daily stats
daily_count = DailyUserStat.count
puts "8️⃣  DailyUserStat records created: #{daily_count}"

if daily_count > 0
  DailyUserStat.all.each do |stat|
    puts "   ✅ Daily stat:"
    puts "      - User: #{stat.user.slug}"
    puts "      - Date: #{stat.date}"
    puts "      - Total clicks: #{stat.total_clicks}"
    puts "      - Unique visitors: #{stat.unique_visitors}"
    puts "      - Top link: #{stat.top_link&.title} (#{stat.top_link_clicks} clicks)"
  end
else
  puts "   ❌ No daily stats created!"
  puts "\n   Debugging - LinkClicks today (#{Date.current}):"
  LinkClick.where(created_at: Date.current.beginning_of_day..Date.current.end_of_day).each do |click|
    puts "      - #{click.created_at.strftime('%H:%M:%S')} | Link #{click.link_id} | User #{click.user_id}"
  end
end
puts ""

puts "=" * 60
puts "✅ All aggregation tests completed!"
