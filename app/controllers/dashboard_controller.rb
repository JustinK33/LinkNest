class DashboardController < ApplicationController
  def show
    @user = Current.session.user
    # Mock links data for front-end display
    @links = [
      { id: 1, title: "My Portfolio", url: "https://example.com/portfolio", visible: true, position: 1 },
      { id: 2, title: "GitHub", url: "https://github.com/username", visible: true, position: 2 },
      { id: 3, title: "Draft Project", url: "https://example.com/draft", visible: false, position: 3 }
    ]

    # Mock analytics data for UI placeholders
    @analytics = {
      total_views: 1234,
      total_clicks: 856,
      views_7d: 89,
      clicks_7d: 54,
      top_links: [
        { title: "My Portfolio", clicks: 342 },
        { title: "GitHub", clicks: 298 },
        { title: "Draft Project", clicks: 12 }
      ]
    }
  end
end
