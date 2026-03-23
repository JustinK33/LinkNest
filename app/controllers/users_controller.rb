class UsersController < ApplicationController
  allow_unauthenticated_access only: %i[ new create ]
  before_action :set_current_user, only: [ :edit, :update ]

  def new
    @user = User.new
  end

  def create
    @user = User.new(user_params)

    begin
      if @user.save
        start_new_session_for(@user)
        redirect_to root_path, notice: "Welcome to LinkNest!"
      else
        render :new, status: :unprocessable_entity
      end
    rescue ActiveRecord::RecordNotUnique
      @user.errors.add(:base, "That username, slug, or email is already in use")
      render :new, status: :unprocessable_entity
    end
  end

  def edit
  end

  def update
    if @user.update(update_user_params)
      redirect_to dashboard_path, notice: "Profile updated successfully!"
    else
      render :edit, status: :unprocessable_entity
    end
  end

  private
    def set_current_user
      @user = Current.session.user
    end

    def user_params
      params.expect(user: [ :username, :first_name, :last_name, :email_address, :bio, :avatar, :password, :password_confirmation ])
    end

    def update_user_params
      params.expect(user: [ :username, :bio, :avatar ])
    end
end
