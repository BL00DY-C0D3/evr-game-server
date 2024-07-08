package wrapper

import "testing"

func Test_parseSessionID(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Test 1",
			args:    args{data: []byte(`{"disc":{"position":[0.0,0.0,0.0],"forward":[0.0,0.0,1.0],"left":[1.0,0.0,0.0],"up":[0.0,1.0,0.0],"velocity":[0.0,0.0,0.0],"bounce_count":0},"orange_team_restart_request":0,"sessionid":"8380FAE5-E20F-4B61-BEDA-B94E0811244F","game_clock_display":"10:00.00","game_status":"pre_match","sessionip":"192.168.56.1","match_type":"Echo_Arena_Private","map_name":"mpl_arena_a","right_shoulder_pressed2":0.0,"teams":[{"team":"BLUE TEAM","possession":false},{"players":[{"name":"sprockee [BOT]","rhand":{"pos":[0.0,-2.5080001,77.559006],"forward":[-0.001,0.001,1.0],"left":[1.0,-0.001,0.001],"up":[0.001,1.0,-0.001]},"playerid":0,"userid":3963667097037078,"is_emote_playing":false,"number":1,"level":1,"stunned":false,"ping":28,"packetlossratio":0.0,"invulnerable":false,"holding_left":"none","possession":false,"head":{"position":[0.0,-2.5080001,77.559006],"forward":[-0.001,0.001,-1.0],"left":[-1.0,-0.001,0.001],"up":[-0.001,1.0,0.001]},"body":{"position":[0.0,-2.5080001,77.559006],"forward":[-0.001,-0.001,-1.0],"left":[-1.0,-0.001,0.001],"up":[-0.001,1.0,-0.001]},"holding_right":"none","lhand":{"pos":[0.0,-2.5080001,77.559006],"forward":[-0.001,0.001,1.0],"left":[1.0,-0.001,0.001],"up":[0.001,1.0,-0.001]},"blocking":false,"velocity":[0.0,0.0,0.0],"stats":{"possession_time":0.0,"points":0,"saves":0,"goals":0,"stuns":0,"passes":0,"catches":0,"steals":0,"blocks":0,"interceptions":0,"assists":0,"shots_taken":0}}],"team":"ORANGE TEAM","possession":false,"stats":{"possession_time":0.0,"points":0,"goals":0,"saves":0,"stuns":0,"interceptions":0,"blocks":0,"passes":0,"catches":0,"steals":0,"assists":0,"shots_taken":0}},{"team":"SPECTATORS","possession":false}],"blue_round_score":0,"orange_points":0,"player":{"vr_left":[1.0,0.0,0.0],"vr_position":[0.0,0.0,0.0],"vr_forward":[0.0,0.0,1.0],"vr_up":[0.0,1.0,0.0]},"private_match":true,"blue_team_restart_request":0,"tournament_match":false,"orange_round_score":0,"rules_changed_by":"[INVALID]","total_round_count":3,"left_shoulder_pressed2":0.0,"left_shoulder_pressed":0.0,"pause":{"paused_state":"unpaused","unpaused_team":"none","paused_requested_team":"none","unpaused_timer":0.0,"paused_timer":0.0},"right_shoulder_pressed":0.0,"blue_points":0,"last_throw":{"arm_speed":0.0,"total_speed":0.0,"off_axis_spin_deg":0.0,"wrist_throw_penalty":0.0,"rot_per_sec":0.0,"pot_speed_from_rot":0.0,"speed_from_arm":0.0,"speed_from_movement":0.0,"speed_from_wrist":0.0,"wrist_align_to_throw_deg":0.0,"throw_align_to_movement_deg":0.0,"off_axis_penalty":0.0,"throw_move_penalty":0.0},"game_clock":600.0,"possession":[-1,-1],"last_score":{"disc_speed":6.6384706e-7,"team":"orange","goal_type":"[NO GOAL]","point_amount":0,"distance_thrown":0.0,"person_scored":"[INVALID]","assist_scored":"[INVALID]"},"rules_changed_at":0,"err_code":0}`)},
			want:    "8380FAE5-E20F-4B61-BEDA-B94E0811244F",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSessionID(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSessionID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseSessionID() = %v, want %v", got, tt.want)
			}
		})
	}
}
