class ProfilesController < ApplicationController
  allow_unauthenticated_access
  before_action :set_user, only: :show

  def show
    @links = @user.links.order(:position)
  end

  private
    def set_user
      @user = User.find_by_slug!(params[:slug])
    rescue ActiveRecord::RecordNotFound
      redirect_to root_path, alert: "Profile not found"
    end
end
