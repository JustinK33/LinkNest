class DashboardController < ApplicationController
  def show
    @user = Current.session.user
    @links = @user.links.order(:position)
  end
end
