class ResetLinkClickCounters < ActiveRecord::Migration[8.1]
  def up
    # Reset counter cache for all links
    # This ensures accuracy after adding counter_cache to LinkClick model
    Link.find_each do |link|
      Link.reset_counters(link.id, :link_clicks)
    end
  end

  def down
    # No need to undo this - the counter cache reset is safe to leave
  end
end
